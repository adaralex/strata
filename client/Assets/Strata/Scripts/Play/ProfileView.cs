using System;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Strata.Location;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The Hero page: a live portrait of the walker's figure, the four stats, what the worn
    /// gear speaks for, the civilization this walker leans to, achievements, and a globe of
    /// every fight this phone has had. Everything is derived from PlayerStore; the portrait
    /// and the globe are small cameras rendering to textures, no art.
    /// </summary>
    public sealed class ProfileView
    {
        private readonly UIKit _ui;
        private readonly PlayerStore _store;
        private readonly Func<StrataAvatar> _avatar;
        private readonly Func<Fix?> _fix;
        private RectTransform _panel;
        private readonly List<UnityEngine.Object> _temp = new List<UnityEngine.Object>();

        public ProfileView(UIKit ui, PlayerStore store, Func<StrataAvatar> avatar, Func<Fix?> fix)
        {
            _ui = ui; _store = store; _avatar = avatar; _fix = fix;
        }

        public void Show(Action closed)
        {
            _panel = _ui.Full("Hero", _ui.Root, UIKit.PanelOpaque);
            _ui.Column(_panel, 40, 16);
            _ui.Header(_panel, "Hero", () => Close(closed));
            var content = _ui.Scroll(_panel);

            // ---- portrait and identity ----
            var top = _ui.Row(content, 520, 20);
            var portrait = _ui.Framed("Portrait", top, UIKit.PanelLight);
            LE(portrait).flexibleWidth = 0.42f;
            var avatar = _avatar();
            if (avatar != null) Portrait(portrait, avatar.transform);
            else _ui.Text(portrait, "no figure on the map yet", 30, false, TextAlignmentOptions.Center, UIKit.Muted);

            var id = _ui.Framed("Identity", top, UIKit.PanelLight);
            LE(id).flexibleWidth = 0.58f;
            _ui.Column(id, 24, 8);
            _ui.Title(id, "WALKER", 22, TextAlignmentOptions.TopLeft, UIKit.Muted);
            _ui.Title(id, Config.StrataSettings.DeviceId.Length > 12 ? Config.StrataSettings.DeviceId.Substring(5, 8).ToUpper() : "WALKER", 40);
            var pref = _store.PreferredCiv();
            _ui.Text(id, pref == null ? "leans to no one yet" : $"leans to {CivPalette.Coloured(pref)}", 32);
            _ui.Text(id, $"{_store.walks} walk{(_store.walks == 1 ? "" : "s")}  ·  {_store.totalMetres / 1000:0.0} km", 30, false, TextAlignmentOptions.TopLeft, UIKit.Muted);
            _ui.Text(id, $"{_store.battle.fights} fights  ·  {_store.battle.wins} won  ·  {_store.battle.drops} drops", 30, false, TextAlignmentOptions.TopLeft, UIKit.Muted);
            var gear = _store.GearCivs();
            if (gear.Count == 0) _ui.Text(id, "wearing nothing", 30, false, TextAlignmentOptions.TopLeft, UIKit.Muted);
            else
            {
                var parts = new List<string>();
                foreach (var kv in gear) parts.Add($"{CivPalette.Coloured(kv.Key)} ×{kv.Value}");
                _ui.Text(id, "gear speaks for " + string.Join("  ", parts), 30);
            }

            // ---- stats ----
            var stats = _ui.Framed("Stats", content, UIKit.PanelLight);
            _ui.Column(stats, 28, 6);
            _ui.Title(stats, "FROM WHAT YOU WEAR", 24);
            var totals = _store.StatTotals();
            var row = _ui.Row(stats, 84, 12);
            foreach (var s in PlayerStore.Stats)
            {
                var cell = _ui.Framed("Stat " + s, row, UIKit.Panel);
                _ui.Column(cell, 12, 0);
                _ui.Title(cell, CivPalette.StatLabel(s).ToUpper(), 22, TextAlignmentOptions.Center, UIKit.Muted);
                _ui.Title(cell, totals.TryGetValue(s, out var v) ? v.ToString("0") : "0", 44, TextAlignmentOptions.Center, UIKit.Ink);
            }

            // ---- ground walked, by civilization ----
            if (_store.metresByCiv.Count > 0)
            {
                var ground = _ui.Framed("Ground", content, UIKit.PanelLight);
                _ui.Column(ground, 28, 6);
                _ui.Title(ground, "GROUND WALKED", 24);
                var sorted = new List<KeyValuePair<string, double>>(_store.metresByCiv);
                sorted.Sort((a, b) => b.Value.CompareTo(a.Value));
                foreach (var kv in sorted)
                    _ui.Text(ground, $"{CivPalette.Coloured(kv.Key)}  {kv.Value / 1000:0.0} km", 32);
            }

            // ---- achievements ----
            var ach = _store.Achievements();
            int earned = 0; foreach (var a in ach) if (a.Earned) earned++;
            _ui.Title(content, $"ACHIEVEMENTS  <color=#AAAAAA>{earned}/{ach.Count}</color>", 24);
            for (int i = 0; i < ach.Count; i += 3)
            {
                var r = _ui.Row(content, 150, 12);
                for (int j = i; j < i + 3; j++)
                {
                    if (j >= ach.Count) { var filler = new GameObject("Filler", typeof(RectTransform)); filler.transform.SetParent(r, false); continue; }
                    var a = ach[j];
                    var tile = _ui.Framed("Ach " + a.Id, r, a.Earned ? UIKit.PanelLight : UIKit.Panel);
                    _ui.Column(tile, 14, 4);
                    _ui.Title(tile, a.Title.ToUpper(), 20, TextAlignmentOptions.Center, a.Earned ? UIKit.Accent : UIKit.Muted);
                    _ui.Text(tile, a.Detail, 24, false, TextAlignmentOptions.Center, a.Earned ? UIKit.Ink : new Color(0.45f, 0.40f, 0.34f));
                }
            }

            // ---- globe ----
            _ui.Title(content, $"WHERE YOU FOUGHT  <color=#AAAAAA>{_store.fightPoints.Count}</color>", 24);
            var globeFrame = _ui.Framed("Globe", content, UIKit.PanelLight);
            LE(globeFrame).minHeight = 720;
            LE(globeFrame).preferredHeight = 720;
            Globe(globeFrame);
        }

        private static LayoutElement LE(RectTransform rt) => rt.GetComponent<LayoutElement>() ?? rt.gameObject.AddComponent<LayoutElement>();

        /// <summary>A camera in front of the figure, rendering into the portrait tile.</summary>
        private void Portrait(RectTransform tile, Transform figure)
        {
            var rt = new RenderTexture(512, 640, 16);
            _temp.Add(rt);
            var camGo = new GameObject("PortraitCam");
            _temp.Add(camGo);
            var cam = camGo.AddComponent<Camera>();
            cam.clearFlags = CameraClearFlags.SolidColor;
            cam.backgroundColor = new Color(0.16f, 0.11f, 0.08f);
            cam.fieldOfView = 30;
            cam.nearClipPlane = 0.5f;
            cam.farClipPlane = 200;
            cam.targetTexture = rt;
            var follow = camGo.AddComponent<PortraitFollow>();
            follow.Target = figure;
            var img = new GameObject("PortraitImage", typeof(RectTransform), typeof(RawImage));
            img.transform.SetParent(tile, false);
            var irt = img.GetComponent<RectTransform>();
            irt.anchorMin = Vector2.zero; irt.anchorMax = Vector2.one;
            irt.offsetMin = new Vector2(8, 8); irt.offsetMax = new Vector2(-8, -8);
            img.GetComponent<RawImage>().texture = rt;
        }

        /// <summary>Keeps a portrait camera in front of a figure that may move and turn.</summary>
        private sealed class PortraitFollow : MonoBehaviour
        {
            public Transform Target;
            private void LateUpdate()
            {
                if (Target == null) return;
                var rig = Target.childCount > 0 ? Target.GetChild(0) : Target;
                var forward = rig.forward; forward.y = 0;
                if (forward.sqrMagnitude < 0.01f) forward = Vector3.back;
                forward.Normalize();
                var chest = Target.position + Vector3.up * 7.5f;
                transform.position = chest + forward * 26f + Vector3.up * 2.5f;
                transform.LookAt(chest);
            }
        }

        /// <summary>A globe far below the map, gridded, with a dot per fight and one for the walker.</summary>
        private void Globe(RectTransform tile)
        {
            var rt = new RenderTexture(768, 768, 16);
            _temp.Add(rt);
            var origin = new Vector3(0, -6000f, 0);
            var globe = GameObject.CreatePrimitive(PrimitiveType.Sphere);
            globe.name = "Globe";
            _temp.Add(globe);
            UnityEngine.Object.Destroy(globe.GetComponent<Collider>());
            globe.transform.position = origin;
            globe.transform.localScale = Vector3.one * 200f;
            Paint(globe.GetComponent<Renderer>(), new Color(0.20f, 0.15f, 0.11f));

            // Graticule every 30 degrees, drawn just above the surface.
            for (int lat = -60; lat <= 60; lat += 30) Ring(globe.transform, lat, true);
            for (int lon = 0; lon < 180; lon += 30) Ring(globe.transform, lon, false);

            // Fights, coloured by civilization; the walker in gold.
            foreach (var p in _store.fightPoints) Dot(globe.transform, p.lat, p.lon, CivPalette.Of(p.civ), p.won ? 1.6f : 1.1f);
            var fix = _fix();
            double cLat = 43.54, cLon = 1.34;
            if (_store.fightPoints.Count > 0) { cLat = 0; cLon = 0; foreach (var p in _store.fightPoints) { cLat += p.lat; cLon += p.lon; } cLat /= _store.fightPoints.Count; cLon /= _store.fightPoints.Count; }
            if (fix.HasValue) { Dot(globe.transform, fix.Value.Lat, fix.Value.Lon, UIKit.Accent, 2.4f); cLat = fix.Value.Lat; cLon = fix.Value.Lon; }

            // Turn the globe so the centre of interest faces the camera, then spin slowly.
            var spin = globe.AddComponent<GlobeSpin>();
            spin.Set((float)cLat, (float)cLon);

            var camGo = new GameObject("GlobeCam");
            _temp.Add(camGo);
            var cam = camGo.AddComponent<Camera>();
            cam.clearFlags = CameraClearFlags.SolidColor;
            cam.backgroundColor = new Color(0.08f, 0.06f, 0.05f);
            cam.fieldOfView = 40;
            cam.nearClipPlane = 1; cam.farClipPlane = 1000;
            cam.transform.position = origin + new Vector3(0, 60f, -300f);
            cam.transform.LookAt(origin);
            cam.targetTexture = rt;

            var img = new GameObject("GlobeImage", typeof(RectTransform), typeof(RawImage), typeof(AspectRatioFitter));
            img.transform.SetParent(tile, false);
            var irt = img.GetComponent<RectTransform>();
            irt.anchorMin = Vector2.zero; irt.anchorMax = Vector2.one;
            irt.offsetMin = new Vector2(8, 8); irt.offsetMax = new Vector2(-8, -8);
            img.GetComponent<RawImage>().texture = rt;
            var fit = img.GetComponent<AspectRatioFitter>();
            fit.aspectMode = AspectRatioFitter.AspectMode.FitInParent;
            fit.aspectRatio = 1f;
        }

        private static Vector3 OnSphere(double lat, double lon, float r)
        {
            double la = lat * Math.PI / 180, lo = lon * Math.PI / 180;
            return new Vector3((float)(Math.Cos(la) * Math.Sin(lo)), (float)Math.Sin(la), (float)(-Math.Cos(la) * Math.Cos(lo))) * r;
        }

        private static void Ring(Transform globe, int deg, bool parallel)
        {
            var go = new GameObject(parallel ? "Parallel " + deg : "Meridian " + deg);
            go.transform.SetParent(globe, false);
            var lr = go.AddComponent<LineRenderer>();
            lr.useWorldSpace = false;
            lr.loop = true;
            lr.widthMultiplier = 1.2f; // world units: the globe is scaled 200×, line widths are not
            lr.material = new Material(Shader.Find("Unlit/Color") ?? Shader.Find("Standard")) { color = UIKit.Accent };
            lr.shadowCastingMode = UnityEngine.Rendering.ShadowCastingMode.Off;
            const int n = 72;
            lr.positionCount = n;
            for (int i = 0; i < n; i++)
            {
                double t = i * 360.0 / n;
                lr.SetPosition(i, parallel ? OnSphere(deg, t, 0.502f) : OnSphere(t - 90, deg, 0.502f));
            }
        }

        private static void Dot(Transform globe, double lat, double lon, Color colour, float size)
        {
            var d = GameObject.CreatePrimitive(PrimitiveType.Sphere);
            d.name = "Fight";
            d.transform.SetParent(globe, false);
            UnityEngine.Object.Destroy(d.GetComponent<Collider>());
            d.transform.localPosition = OnSphere(lat, lon, 0.505f);
            d.transform.localScale = Vector3.one * size * 0.01f;
            Paint(d.GetComponent<Renderer>(), colour);
        }

        private static void Paint(Renderer r, Color colour)
        {
            r.material = new Material(Shader.Find("Unlit/Color") ?? Shader.Find("Standard")) { color = colour };
            r.shadowCastingMode = UnityEngine.Rendering.ShadowCastingMode.Off;
            r.receiveShadows = false;
        }

        private sealed class GlobeSpin : MonoBehaviour
        {
            private float _lon, _lat, _t;
            public void Set(float lat, float lon) { _lat = lat; _lon = lon; }
            private void Update()
            {
                _t += Time.deltaTime;
                // Bring the point of interest to the camera-facing side (-z), then drift east and west.
                float drift = Mathf.Sin(_t * 0.25f) * 25f;
                transform.rotation = Quaternion.Euler(-_lat * 0.8f, -_lon + drift, 0);
            }
        }

        private void Close(Action closed)
        {
            foreach (var o in _temp) if (o != null) UnityEngine.Object.Destroy(o);
            _temp.Clear();
            if (_panel != null) UnityEngine.Object.Destroy(_panel.gameObject);
            _panel = null;
            closed?.Invoke();
        }
    }
}
