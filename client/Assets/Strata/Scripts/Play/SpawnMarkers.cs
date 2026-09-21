using System;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using Strata.Map;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// World-space markers for spawns, drawn to read on the parchment map: an ink ring on the
    /// ground, an unlit pillar in the civilization colour, sized by rank, with a floating
    /// label. Commons hide until the walker is close: beyond StirM nothing shows, between
    /// StirM and RevealM a faint pulse on the ground says something is there, inside RevealM
    /// the figure springs up and OnReveal fires (the ambush). Elites and bosses are visible
    /// from anywhere; a boss carries a tall beacon. Tap detection is a raycast against the
    /// marker's colliders, hidden markers excluded. No art in phase 0.
    /// </summary>
    public sealed class SpawnMarkers
    {
        public const float RevealM = 45f;
        public const float StirM = 90f;
        public const float HideAgainM = 120f;

        /// <summary>A common just revealed itself: the spawn and its distance in metres.</summary>
        public event Action<Spawn, double> OnReveal;

        private readonly MapAdapter _map;
        private readonly Transform _root;
        private readonly Dictionary<string, Marker> _markers = new Dictionary<string, Marker>();
        private readonly Camera _camera;
        private double _playerLat, _playerLon;
        private bool _havePlayer;

        private static readonly Color Ink = new Color(0.37f, 0.20f, 0.20f);     // matches the road outline ink
        private static readonly Color Paper = new Color(0.93f, 0.85f, 0.66f);   // parchment highlight
        private static readonly Color Gold = new Color(0.85f, 0.68f, 0.35f);

        private sealed class Marker
        {
            public Spawn Spawn;
            public GameObject Go;
            public Transform Figure;   // ring, pillar, cap, label: what a hidden common does not show
            public Renderer Stir;      // the pulse on the ground while it is close but unseen
            public TextMeshPro Label;
            public bool AlwaysVisible;
            public bool Revealed;
            public float RevealT;
        }

        public SpawnMarkers(MapAdapter map, Camera camera)
        {
            _map = map;
            _camera = camera;
            _root = new GameObject("SpawnMarkers").transform;
        }

        /// <summary>Where the walker is, for reveal distances. Call on every fix.</summary>
        public void SetPlayer(double lat, double lon)
        {
            _playerLat = lat; _playerLon = lon; _havePlayer = true;
        }

        /// <summary>Whether a spawn currently shows on the map (always for elites and bosses).</summary>
        public bool IsVisible(string id) => _markers.TryGetValue(id, out var m) && (m.AlwaysVisible || m.Revealed);

        public void Sync(List<Spawn> spawns)
        {
            var seen = new HashSet<string>();
            foreach (var sp in spawns)
            {
                seen.Add(sp.Id);
                if (!_markers.TryGetValue(sp.Id, out var m))
                {
                    m = Create(sp);
                    _markers[sp.Id] = m;
                }
                m.Spawn = sp;
            }
            var gone = new List<string>();
            foreach (var kv in _markers)
                if (!seen.Contains(kv.Key)) gone.Add(kv.Key);
            foreach (var id in gone)
            {
                UnityEngine.Object.Destroy(_markers[id].Go);
                _markers.Remove(id);
            }
        }

        public void Remove(string id)
        {
            if (_markers.TryGetValue(id, out var m))
            {
                UnityEngine.Object.Destroy(m.Go);
                _markers.Remove(id);
            }
        }

        private static float ScaleFor(Spawn sp) => sp.Rank == "boss" ? 2.0f : sp.Rank == "elite" ? 1.4f : 1f;

        private Marker Create(Spawn sp)
        {
            float s = ScaleFor(sp);
            var civ = CivPalette.Of(sp.Civ);
            bool boss = sp.Rank == "boss", elite = sp.Rank == "elite";

            var go = new GameObject("spawn " + sp.Name);
            go.transform.SetParent(_root, false);
            go.AddComponent<MarkerTag>().SpawnId = sp.Id;
            // A generous invisible tap target around the whole marker.
            var hitbox = go.AddComponent<SphereCollider>();
            hitbox.center = new Vector3(0, 3.5f * s, 0);
            hitbox.radius = 5f * s;

            // The pulse on the ground for a common that is near but unseen.
            var stir = Disc(go.transform, "Stir", 6f, 0.08f, new Color(Ink.r, Ink.g, Ink.b, 0.35f), true);
            UnityEngine.Object.Destroy(stir.GetComponent<Collider>());
            stir.SetActive(false);

            var figure = new GameObject("Figure").transform;
            figure.SetParent(go.transform, false);

            // Ink ring on the ground: a dark disc with a lighter disc inset above it.
            Disc(figure, "Ring", 9f * s, 0.05f, Ink);
            Disc(figure, "RingInner", 7f * s, 0.12f, Paper);
            Disc(figure, "RingCore", 4f * s, 0.18f, civ);

            // Pillar in the civilization colour, unlit so it stays bright in fog and shade.
            var body = GameObject.CreatePrimitive(PrimitiveType.Capsule);
            body.name = "Body";
            body.transform.SetParent(figure, false);
            body.transform.localScale = new Vector3(2.4f * s, 3.2f * s, 2.4f * s);
            body.transform.localPosition = new Vector3(0, 3.2f * s, 0);
            Paint(body.GetComponent<Renderer>(), civ);

            // Cap: ink for commons, gold for elites and bosses, so rank reads at a glance.
            var cap = GameObject.CreatePrimitive(PrimitiveType.Sphere);
            cap.name = "Cap";
            cap.transform.SetParent(figure, false);
            cap.transform.localScale = new Vector3(1.4f * s, 1.4f * s, 1.4f * s);
            cap.transform.localPosition = new Vector3(0, 6.2f * s, 0);
            UnityEngine.Object.Destroy(cap.GetComponent<Collider>());
            Paint(cap.GetComponent<Renderer>(), elite || boss ? Gold : Ink);

            if (boss)
            {
                // A tall translucent beacon in the civilization colour, seen from across town.
                var beacon = GameObject.CreatePrimitive(PrimitiveType.Cylinder);
                beacon.name = "Beacon";
                beacon.transform.SetParent(figure, false);
                beacon.transform.localScale = new Vector3(1.6f, 40f, 1.6f);
                beacon.transform.localPosition = new Vector3(0, 40f, 0);
                UnityEngine.Object.Destroy(beacon.GetComponent<Collider>());
                Paint(beacon.GetComponent<Renderer>(), new Color(civ.r, civ.g, civ.b, 0.35f), true);
            }

            var labelGo = new GameObject("Label");
            labelGo.transform.SetParent(figure, false);
            labelGo.transform.localPosition = new Vector3(0, 7.5f * s, 0);
            var label = labelGo.AddComponent<TextMeshPro>();
            label.text = LabelFor(sp);
            label.fontSize = boss ? 11 : 8;
            label.alignment = TextAlignmentOptions.Center;
            label.color = Paper;
            label.outlineWidth = 0.25f;
            label.outlineColor = Ink;

            var m = new Marker { Spawn = sp, Go = go, Figure = figure, Stir = stir.GetComponent<Renderer>(), Label = label, AlwaysVisible = boss || elite };
            if (!m.AlwaysVisible) figure.gameObject.SetActive(false);
            return m;
        }

        private static GameObject Disc(Transform parent, string name, float diameter, float y, Color colour, bool translucent = false)
        {
            var d = GameObject.CreatePrimitive(PrimitiveType.Cylinder);
            d.name = name;
            d.transform.SetParent(parent, false);
            d.transform.localScale = new Vector3(diameter, 0.04f, diameter);
            d.transform.localPosition = new Vector3(0, y, 0);
            Paint(d.GetComponent<Renderer>(), colour, translucent);
            return d;
        }

        private static void Paint(Renderer r, Color colour, bool translucent = false)
        {
            bool srp = UnityEngine.Rendering.GraphicsSettings.currentRenderPipeline != null;
            Shader shader = null;
            if (translucent) shader = Shader.Find("GoMap/TransparentShader") ?? Shader.Find("Legacy Shaders/Transparent/Diffuse");
            if (shader == null) shader = (srp ? Shader.Find("Universal Render Pipeline/Unlit") : null) ?? Shader.Find("Unlit/Color") ?? Shader.Find("Standard");
            r.material = new Material(shader);
            r.material.color = colour;
            r.shadowCastingMode = UnityEngine.Rendering.ShadowCastingMode.Off;
            r.receiveShadows = false;
        }

        private static string LabelFor(Spawn sp)
        {
            var rank = sp.Rank == "common" ? "" : sp.Rank.ToUpper() + " ";
            var hybrid = string.IsNullOrEmpty(sp.Secondary) ? "" : $"\n<size=70%>with {CivPalette.Label(sp.Secondary)}</size>";
            return $"{rank}{sp.Name}\n<size=75%>{CivPalette.Label(sp.Civ)} · {sp.TierName}</size>{hybrid}";
        }

        /// <summary>Reposition every marker, run the hide/stir/reveal states, face labels to the camera.</summary>
        public void Update()
        {
            float dt = Time.deltaTime;
            foreach (var m in _markers.Values)
            {
                m.Go.transform.position = _map.WorldPosition(m.Spawn.Lat, m.Spawn.Lon);

                if (!m.AlwaysVisible && _havePlayer)
                {
                    double d = Geo.DistanceM(_playerLat, _playerLon, m.Spawn.Lat, m.Spawn.Lon);
                    if (!m.Revealed)
                    {
                        if (d <= RevealM)
                        {
                            m.Revealed = true;
                            m.RevealT = 0f;
                            m.Figure.gameObject.SetActive(true);
                            m.Stir.gameObject.SetActive(false);
                            OnReveal?.Invoke(m.Spawn, d);
                        }
                        else
                        {
                            bool stir = d <= StirM;
                            if (m.Stir.gameObject.activeSelf != stir) m.Stir.gameObject.SetActive(stir);
                            if (stir)
                            {
                                float pulse = 0.5f + 0.5f * Mathf.Sin(Time.time * 3f + m.Spawn.Id.GetHashCode() * 0.001f);
                                m.Stir.transform.localScale = new Vector3(4f + 4f * pulse, 0.04f, 4f + 4f * pulse);
                                var c = m.Stir.material.color; c.a = 0.15f + 0.35f * pulse; m.Stir.material.color = c;
                            }
                        }
                    }
                    else if (d > HideAgainM)
                    {
                        m.Revealed = false;
                        m.Figure.gameObject.SetActive(false);
                    }
                    else if (m.RevealT < 0.45f)
                    {
                        m.RevealT += dt;
                        float t = Mathf.Clamp01(m.RevealT / 0.45f);
                        float k = 1f + 0.35f * Mathf.Sin(t * Mathf.PI); // overshoot then settle
                        m.Figure.localScale = Vector3.one * Mathf.Lerp(0.05f, 1f, t) * k;
                    }
                }

                if (_camera != null && m.Figure.gameObject.activeSelf)
                {
                    var fwd = m.Label.transform.position - _camera.transform.position;
                    fwd.y = 0;
                    if (fwd.sqrMagnitude > 0.001f) m.Label.transform.rotation = Quaternion.LookRotation(fwd);
                }
            }
        }

        /// <summary>The spawn under a screen point, if any; hidden commons cannot be tapped.</summary>
        public Spawn Hit(Vector2 screen)
        {
            if (_camera == null) return null;
            var ray = _camera.ScreenPointToRay(screen);
            // RaycastAll: GO Map buildings and tiles carry colliders too and may
            // sit in front of a marker from the camera's angle.
            Spawn best = null;
            float bestD = float.MaxValue;
            foreach (var hit in Physics.RaycastAll(ray, 5000f))
            {
                var tag = hit.collider.GetComponentInParent<MarkerTag>();
                if (tag != null && hit.distance < bestD && _markers.TryGetValue(tag.SpawnId, out var m) && (m.AlwaysVisible || m.Revealed))
                {
                    best = m.Spawn;
                    bestD = hit.distance;
                }
            }
            return best;
        }

        public sealed class MarkerTag : MonoBehaviour
        {
            public string SpawnId;
        }
    }
}
