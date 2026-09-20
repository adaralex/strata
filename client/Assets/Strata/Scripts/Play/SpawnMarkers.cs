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
    /// World-space markers for spawns: a capsule coloured by civilization, sized by rank,
    /// with a floating label. No art in phase 0. Tap detection is a raycast against the
    /// capsule collider.
    /// </summary>
    public sealed class SpawnMarkers
    {
        private readonly MapAdapter _map;
        private readonly Transform _root;
        private readonly Dictionary<string, Marker> _markers = new Dictionary<string, Marker>();
        private readonly Camera _camera;

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

        private Marker Create(Spawn sp)
        {
            var go = GameObject.CreatePrimitive(PrimitiveType.Capsule);
            go.name = "spawn " + sp.Name;
            go.transform.SetParent(_root, false);
            float scale = sp.Rank == "boss" ? 2.2f : sp.Rank == "elite" ? 1.5f : 1f;
            go.transform.localScale = new Vector3(2f * scale, 3f * scale, 2f * scale);
            var r = go.GetComponent<Renderer>();
            r.material = new Material(Shader.Find("Universal Render Pipeline/Lit") ?? Shader.Find("Standard"));
            r.material.color = CivPalette.Of(sp.Civ);
            var tag = go.AddComponent<MarkerTag>();
            tag.SpawnId = sp.Id;
            // A generous invisible tap target around the capsule.
            var hitbox = go.AddComponent<SphereCollider>();
            hitbox.radius = 1.2f;

            var labelGo = new GameObject("Label");
            labelGo.transform.SetParent(go.transform, false);
            labelGo.transform.localPosition = new Vector3(0, 1.4f, 0);
            var label = labelGo.AddComponent<TextMeshPro>();
            label.text = LabelFor(sp);
            label.fontSize = 6;
            label.alignment = TextAlignmentOptions.Center;
            label.color = Color.white;
            label.outlineWidth = 0.2f;
            label.outlineColor = Color.black;
            return new Marker { Spawn = sp, Go = go, Label = label };
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
                var pos = _map.WorldPosition(m.Spawn.Lat, m.Spawn.Lon);
                m.Go.transform.position = pos + Vector3.up * (1.5f * m.Go.transform.localScale.y / 3f);
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
