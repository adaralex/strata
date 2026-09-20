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
    /// ground, an unlit pillar in the civilization colour (so fog and lighting never dim it),
    /// sized by rank, with a floating label. No art in phase 0. Tap detection is a raycast
    /// against the marker's colliders.
    /// </summary>
    public sealed class SpawnMarkers
    {
        private readonly MapAdapter _map;
        private readonly Transform _root;
        private readonly Dictionary<string, Marker> _markers = new Dictionary<string, Marker>();
        private readonly Camera _camera;

        private static readonly Color Ink = new Color(0.37f, 0.20f, 0.20f);     // matches the road outline ink
        private static readonly Color Paper = new Color(0.93f, 0.85f, 0.66f);   // parchment highlight

        private sealed class Marker
        {
            public Spawn Spawn;
            public GameObject Go;
            public TextMeshPro Label;
        }

        public SpawnMarkers(MapAdapter map, Camera camera)
        {
            _map = map;
            _camera = camera;
            _root = new GameObject("SpawnMarkers").transform;
        }

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

            var go = new GameObject("spawn " + sp.Name);
            go.transform.SetParent(_root, false);
            go.AddComponent<MarkerTag>().SpawnId = sp.Id;
            // A generous invisible tap target around the whole marker.
            var hitbox = go.AddComponent<SphereCollider>();
            hitbox.center = new Vector3(0, 3.5f * s, 0);
            hitbox.radius = 5f * s;

            // Ink ring on the ground: a dark disc with a lighter disc inset above it.
            Disc(go.transform, "Ring", 9f * s, 0.05f, Ink);
            Disc(go.transform, "RingInner", 7f * s, 0.12f, Paper);
            Disc(go.transform, "RingCore", 4f * s, 0.18f, civ);

            // Pillar in the civilization colour, unlit so it stays bright in fog and shade.
            var body = GameObject.CreatePrimitive(PrimitiveType.Capsule);
            body.name = "Body";
            body.transform.SetParent(go.transform, false);
            body.transform.localScale = new Vector3(2.4f * s, 3.2f * s, 2.4f * s);
            body.transform.localPosition = new Vector3(0, 3.2f * s, 0);
            Paint(body.GetComponent<Renderer>(), civ);

            // Ink cap on top of the pillar so it still reads as drawn when the colour is pale.
            var cap = GameObject.CreatePrimitive(PrimitiveType.Sphere);
            cap.name = "Cap";
            cap.transform.SetParent(go.transform, false);
            cap.transform.localScale = new Vector3(1.4f * s, 1.4f * s, 1.4f * s);
            cap.transform.localPosition = new Vector3(0, 6.2f * s, 0);
            UnityEngine.Object.Destroy(cap.GetComponent<Collider>());
            Paint(cap.GetComponent<Renderer>(), Ink);

            var labelGo = new GameObject("Label");
            labelGo.transform.SetParent(go.transform, false);
            labelGo.transform.localPosition = new Vector3(0, 7.5f * s, 0);
            var label = labelGo.AddComponent<TextMeshPro>();
            label.text = LabelFor(sp);
            label.fontSize = 8;
            label.alignment = TextAlignmentOptions.Center;
            label.color = Paper;
            label.outlineWidth = 0.25f;
            label.outlineColor = Ink;
            return new Marker { Spawn = sp, Go = go, Label = label };
        }

        private static void Disc(Transform parent, string name, float diameter, float y, Color colour)
        {
            var d = GameObject.CreatePrimitive(PrimitiveType.Cylinder);
            d.name = name;
            d.transform.SetParent(parent, false);
            d.transform.localScale = new Vector3(diameter, 0.04f, diameter);
            d.transform.localPosition = new Vector3(0, y, 0);
            Paint(d.GetComponent<Renderer>(), colour);
        }

        private static void Paint(Renderer r, Color colour)
        {
            bool srp = UnityEngine.Rendering.GraphicsSettings.currentRenderPipeline != null;
            var shader = (srp ? Shader.Find("Universal Render Pipeline/Unlit") : null) ?? Shader.Find("Unlit/Color") ?? Shader.Find("Standard");
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

        /// <summary>Reposition every marker and face labels to the camera.</summary>
        public void Update()
        {
            foreach (var m in _markers.Values)
            {
                m.Go.transform.position = _map.WorldPosition(m.Spawn.Lat, m.Spawn.Lon);
                if (_camera != null)
                {
                    var fwd = m.Label.transform.position - _camera.transform.position;
                    fwd.y = 0;
                    if (fwd.sqrMagnitude > 0.001f) m.Label.transform.rotation = Quaternion.LookRotation(fwd);
                }
            }
        }

        /// <summary>The spawn under a screen point, if any.</summary>
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
                if (tag != null && hit.distance < bestD && _markers.TryGetValue(tag.SpawnId, out var m))
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
