using System;
using System.Collections;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using Strata.Config;
using Strata.Location;
using Strata.Map;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The walk test, orchestrated. Put this on an empty GameObject with a MapAdapter and
    /// assign the map adapter and the camera. Everything else is built at runtime.
    ///
    /// Flow: fixes arrive; the ground strip refreshes when the cell changes or every 30 s;
    /// spawns refresh after 50 m, on the epoch turning, or every 30 s; a tap on a marker
    /// within range opens the fight; a win posts the collapse and the server's item is
    /// shown. The client never decides anything valuable.
    /// </summary>
    public sealed class WalkController : MonoBehaviour
    {
        public StrataSettings settings;
        public MapAdapter map;
        public Camera worldCamera;

        private WorldClient _client;
        private UIKit _ui;
        private GroundStrip _strip;
        private SpawnMarkers _markers;
        private ServiceMarkers _services;
        private Fight _fight;
        private ItemCard _card;
        private Debrief _debrief;
        private PlayerStore _store;
        private EquipmentView _equipment;
        private InventoryView _inventory;
        private CodexView _codex;
        private FixSource _fixes;
        private readonly WalkLog _log = new WalkLog();

        private LookupOut _lookup;
        private SpawnsOut _spawns;
        // Spawns this device has already collapsed. worldd keeps listing them until the epoch
        // turns, so without this the marker comes back on the next refresh and a second tap is
        // refused as "already collapsed by this device".
        private readonly HashSet<string> _collapsed = new HashSet<string>();
        private Fix? _fix;
        private double _lastSpawnLat, _lastSpawnLon;
        private float _lastLookupAt = -999, _lastSpawnsAt = -999;
        private bool _busy, _modal;
        private string _status = "";
        private TextMeshProUGUI _statusText;

        private void Awake()
        {
            if (settings == null) settings = StrataSettings.Defaults();
            if (worldCamera == null) worldCamera = Camera.main;
            if (map == null) map = GetComponent<MapAdapter>() ?? gameObject.AddComponent<MapAdapter>();
            _client = new WorldClient(settings.EffectiveBaseUrl, StrataSettings.DeviceId);
            _log.device = StrataSettings.DeviceId;

            _ui = new UIKit();
            _strip = new GroundStrip(_ui);
            _markers = new SpawnMarkers(map, worldCamera);
            _services = new ServiceMarkers();
            map.ConfigureServices(ServiceMarkers.Kinds, _services.Template, _services.OnPoi);
            _fight = new Fight(_ui);
            _card = new ItemCard(_ui);
            _debrief = new Debrief(_ui);
            _store = PlayerStore.Load();
            _equipment = new EquipmentView(_ui, _store);
            _inventory = new InventoryView(_ui, _store);
            _codex = new CodexView(_ui, _store);
            BuildBottomBar();
            UI.TapCatcher.Create(_ui.Root).OnTap += OnTap;

            _fixes = gameObject.AddComponent<FixSource>();
            _fixes.intervalSeconds = settings.fixIntervalSeconds;
            _fixes.fallback = () => map.TryGetMapLocation(out var lat, out var lon) ? (true, lat, lon) : (false, 0.0, 0.0);
            _fixes.OnFix += OnFix;
            _fixes.OnStatus += s => SetStatus(s);
#if UNITY_EDITOR
            _sim = gameObject.AddComponent<WalkSimulator>();
            _sim.Bind(this, map);
#endif
        }

        private void Start()
        {
            SetStatus($"worldd {_client.BaseUrl}  device {_client.DeviceId.Substring(0, 12)}…");
            StartCoroutine(_client.Health(r => SetStatus(r.Ok ? $"connected to {_client.BaseUrl}" : $"worldd unreachable at {_client.BaseUrl}: {r.Error}")));
            _fixes.Begin();
        }

        private void Update()
        {
            _markers.Update();
            _strip.Tick();
            if (_fight.Active) _fight.Update(Time.deltaTime);
            if (_fix.HasValue && !_busy)
            {
                var now = Time.time;
                if (now - _lastLookupAt > settings.refreshIntervalSeconds) StartCoroutine(RefreshLookup());
                if (now - _lastSpawnsAt > settings.refreshIntervalSeconds) StartCoroutine(RefreshSpawns());
                if (_spawns != null && DateTimeOffset.UtcNow.ToUnixTimeSeconds() >= _spawns.EpochEnds) StartCoroutine(RefreshSpawns());
            }
        }

#if UNITY_EDITOR
        private WalkSimulator _sim;

        // ---- editor walk emulation hooks (never in the phone build) ----
        public bool FightActive => _fight.Active;
        public bool Modal => _modal;
        public void AutoResolveFight() => _fight.AutoResolve();
        public void SimNote(string s) { if (!_modal) SetStatus(s); Debug.Log("[walk emulation] " + s); }

        /// <summary>Tap the nearest spawn within interaction range, if any. Returns whether one was engaged.</summary>
        public bool EngageNearestInRange()
        {
            if (_modal || _fight.Active || _spawns == null || !_fix.HasValue) return false;
            if (_spawns.SpeedGated || (_lookup != null && _lookup.Excluded)) return false;
            var f = _fix.Value;
            Spawn best = null; double bestD = double.MaxValue;
            foreach (var sp in _spawns.Spawns)
            {
                var d = Geo.DistanceM(f.Lat, f.Lon, sp.Lat, sp.Lon);
                if (d < bestD) { bestD = d; best = sp; }
            }
            if (best == null || bestD > settings.interactionRangeM) return false;
            TryEngage(best);
            return _fight.Active;
        }

        /// <summary>Press "Keep walking" on an open item card, or "Back to the map" on a debrief.</summary>
        public void DismissCard()
        {
            foreach (var b in _ui.Root.GetComponentsInChildren<UnityEngine.UI.Button>(true))
                if (b.name == "Button Keep walking" || b.name == "Button Back to the map") { b.onClick.Invoke(); return; }
        }
#endif

        private void OnTap(Vector2 screen)
        {
            if (_modal || _fight.Active) return;
            var sp = _markers.Hit(screen);
            if (sp != null) TryEngage(sp);
            else SetStatus(NearestHint() ?? "tap a marker to engage it");
        }

        /// <summary>"nearest: Bog-Wight, 120 m" from the cached spawns and the last fix.</summary>
        private string NearestHint()
        {
            if (_spawns == null || _spawns.Spawns.Count == 0 || !_fix.HasValue) return null;
            var f = _fix.Value;
            Spawn best = null;
            double bestD = double.MaxValue;
            foreach (var sp in _spawns.Spawns)
            {
                var d = Geo.DistanceM(f.Lat, f.Lon, sp.Lat, sp.Lon);
                if (d < bestD) { bestD = d; best = sp; }
            }
            return best == null ? null : $"nearest: {best.Name}, {bestD:0} m{(bestD <= settings.interactionRangeM ? "  TAP IT" : "")}";
        }

        private void OnFix(Fix f)
        {
            bool first = !_fix.HasValue;
            _fix = f;
            _log.Fix(f.Lat, f.Lon);
            if (!_modal && !_fight.Active) { var hint = NearestHint(); if (hint != null) SetStatus(hint); }
            if (first || _lookup == null)
            {
                StartCoroutine(RefreshLookup());
                StartCoroutine(RefreshSpawns());
                return;
            }
            var moved = Geo.DistanceM(_lastSpawnLat, _lastSpawnLon, f.Lat, f.Lon);
            if (moved > settings.spawnRefreshDistanceM) StartCoroutine(RefreshSpawns());
            // A new r8 cell is roughly 460 m; refresh the ground when we have moved that far
            // since the last lookup rather than tracking cells client side.
            if (moved > 200 && Time.time - _lastLookupAt > 5f) StartCoroutine(RefreshLookup());
        }

        private IEnumerator RefreshLookup()
        {
            if (!_fix.HasValue) yield break;
            _lastLookupAt = Time.time;
            var f = _fix.Value;
            yield return _client.Lookup(f.Lat, f.Lon, r =>
            {
                if (!r.Ok) { SetStatus("lookup: " + r.Error); return; }
                _lookup = r.Value;
                _log.SetGround(_lookup);
                _strip.Show(_lookup, _ui);
                UpdateBanner();
            });
        }

        private IEnumerator RefreshSpawns()
        {
            if (!_fix.HasValue) yield break;
            _lastSpawnsAt = Time.time;
            var f = _fix.Value;
            _lastSpawnLat = f.Lat;
            _lastSpawnLon = f.Lon;
            yield return _client.Spawns(f.Lat, f.Lon, r =>
            {
                if (!r.Ok) { SetStatus("spawns: " + r.Error); return; }
                _spawns = r.Value;
                _spawns.Spawns.RemoveAll(s => _collapsed.Contains(s.Id));
                _strip.SetEpochEnds(_spawns.EpochEnds);
                _markers.Sync(_spawns.Spawns);
                UpdateBanner();
                SetStatus($"{_spawns.Spawns.Count} nearby · epoch {_spawns.Epoch} · {_spawns.Cond?.Phase}");
            });
        }

        private void UpdateBanner()
        {
            if (_spawns != null && _spawns.SpeedGated) _strip.Banner("passive mode: moving too fast to interact");
            else if (_lookup != null && _lookup.Excluded) _strip.Banner($"no interaction here: {_lookup.Zone?.Kind} {_lookup.Zone?.Name}".TrimEnd());
            else _strip.Banner(null);
        }

        private void TryEngage(Spawn sp)
        {
            if (!_fix.HasValue) return;
            if (_spawns != null && _spawns.SpeedGated) { SetStatus("moving too fast to interact"); return; }
            if (_lookup != null && _lookup.Excluded) { SetStatus("no interaction here"); return; }
            var f = _fix.Value;
            var d = Geo.DistanceM(f.Lat, f.Lon, sp.Lat, sp.Lon);
            if (d > settings.interactionRangeM)
            {
                SetStatus($"{sp.Name} is {d:0} m away; walk within {settings.interactionRangeM:0} m");
                return;
            }
            _modal = true;
            _fight.Begin(sp, (won, auto) =>
            {
                _log.Fought(won, auto);
                _store.RecordFight(sp, won, auto);
                if (!won) { _modal = false; SetStatus($"{sp.Name} slipped away"); return; }
                StartCoroutine(Collapse(sp));
            });
        }

        private IEnumerator Collapse(Spawn sp)
        {
            var f = _fix.Value;
            var body = new CollapseIn { SpawnId = sp.Id, Cell = sp.Cell, Epoch = sp.Epoch, Lat = f.Lat, Lon = f.Lon };
            SetStatus("the server is deciding the drop…");
            yield return _client.Collapse(body, r =>
            {
                if (!r.Ok)
                {
                    _modal = false;
                    SetStatus($"collapse refused ({r.Status}): {r.Error}");
                    if (r.Status == 409) Forget(sp); // already ours: never offer it again
                    return;
                }
                Forget(sp);
                _log.Received(r.Value.Item);
                _store.AddItem(r.Value.Item);
                _card.Show(r.Value, () => { _modal = false; });
                SetStatus($"{r.Value.Item.Name} is yours");
            });
        }

        /// <summary>Drop a spawn from the map and the cached list for the rest of the session.</summary>
        private void Forget(Spawn sp)
        {
            _collapsed.Add(sp.Id);
            _markers.Remove(sp.Id);
            _spawns?.Spawns.RemoveAll(s => s.Id == sp.Id);
        }

        private void BuildBottomBar()
        {
            var bar = _ui.Bar("BottomBar", _ui.Root, UIKit.Panel, false, 310);
            var col = _ui.Column(bar, 24, 10);
            col.childAlignment = TextAnchor.LowerLeft;
            _statusText = _ui.Text(bar, "", 32, false, TextAlignmentOptions.BottomLeft, new Color(0.75f, 0.75f, 0.75f));
            var views = _ui.Row(bar, 90, 16);
            _ui.Button_(views, "Gear", UIKit.PanelLight, () => OpenView(_equipment.Show), 90, TextAlignmentOptions.Center, UIKit.Ink);
            _ui.Button_(views, "Bag", UIKit.PanelLight, () => OpenView(_inventory.Show), 90, TextAlignmentOptions.Center, UIKit.Ink);
            _ui.Button_(views, "Codex", UIKit.PanelLight, () => OpenView(_codex.Show), 90, TextAlignmentOptions.Center, UIKit.Ink);
            var row = new GameObject("Row", typeof(RectTransform), typeof(UnityEngine.UI.HorizontalLayoutGroup), typeof(UnityEngine.UI.LayoutElement));
            row.transform.SetParent(bar, false);
            var h = row.GetComponent<UnityEngine.UI.HorizontalLayoutGroup>();
            h.spacing = 16;
            h.childForceExpandWidth = true;
            row.GetComponent<UnityEngine.UI.LayoutElement>().minHeight = 90;
            var rrt = row.GetComponent<RectTransform>();
            _ui.Button_(rrt, "End walk", UIKit.Accent, () => { if (_modal) return; _modal = true; _debrief.Show(_log, () => _modal = false); }, 90);
            _ui.Button_(rrt, "Server", UIKit.PanelLight, ShowSettings, 90);
        }

        private void OpenView(Action<Action> show)
        {
            if (_modal) return;
            _modal = true;
            show(() => _modal = false);
        }

        private void ShowSettings()
        {
            if (_modal) return;
            _modal = true;
            var panel = _ui.Full("Settings", _ui.Root, UIKit.Panel);
            _ui.Column(panel, 40, 20);
            _ui.Title(panel, "Server", 52);
            _ui.Text(panel, "worldd base URL", 36, false, TextAlignmentOptions.TopLeft, UIKit.Muted);
            var input = _ui.Input(panel, _client.BaseUrl, "http://host:8080");
            _ui.Text(panel, $"device id {_client.DeviceId}\nanonymous, generated on this phone, not an account", 30, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            _ui.Text(panel, map.UsingGoMap ? "map: GO Map" : "map: flat fallback (STRATA_GOMAP not defined)", 30, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            _ui.Button_(panel, "Save and reconnect", UIKit.Accent, () =>
            {
                settings.SaveBaseUrl(input.text);
                _client.BaseUrl = settings.EffectiveBaseUrl;
                _lookup = null;
                _spawns = null;
                _lastLookupAt = _lastSpawnsAt = -999;
                Destroy(panel.gameObject);
                _modal = false;
                StartCoroutine(_client.Health(r => SetStatus(r.Ok ? $"connected to {_client.BaseUrl}" : $"unreachable: {r.Error}")));
            });
#if UNITY_EDITOR
            _ui.Button_(panel, _sim != null && _sim.Running ? "Stop walk emulation" : "Emulate the walk", UIKit.PanelLight, () =>
            {
                Destroy(panel.gameObject);
                _modal = false;
                _sim?.Toggle();
            }, 90);
#endif
            _ui.Button_(panel, "Cancel", UIKit.PanelLight, () => { Destroy(panel.gameObject); _modal = false; }, 90);
        }

        private void SetStatus(string s)
        {
            _status = s;
            if (_statusText != null) _statusText.text = s;
        }
    }
}
