using System.Collections.Generic;
using TMPro;
using UnityEngine;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// City services drawn on the map so a tester knows where to head (PLAN §9): an ink
    /// building with a sign naming the game's function, hearth, vault, forge, apothecary and
    /// so on. Phase 0 visuals only: the points come from the map tiles' POI layer through
    /// MapAdapter, nothing is tapped and no effect is applied. The server's poi_classes.json
    /// stays the authority on what a service is; this table only mirrors it for the tiles'
    /// kind names, and leaves out memorials, which need human review before they count.
    /// </summary>
    public sealed class ServiceMarkers
    {
        /// <summary>Game service -> the tile POI kinds (GO Map names) that show it.</summary>
        public static readonly Dictionary<string, string[]> Kinds = new Dictionary<string, string[]>
        {
            { "hearth", new[] { "bakery", "convenience", "supermarket", "greengrocer", "butcher", "cafe", "fast_food" } },
            { "vault", new[] { "bank", "post_office", "atm" } },
            { "wardrobe", new[] { "clothes", "jewelry", "tailor", "department_store" } },
            { "forge", new[] { "hardware", "doityourself", "trade" } },
            { "apothecary", new[] { "pharmacy" } },
            { "scriptorium", new[] { "library", "university", "college", "books", "information" } },
            { "beacon", new[] { "museum", "gallery", "monument", "ruins", "archaeological_site" } },
            { "spring", new[] { "drinking_water", "spring", "water_well" } },
            { "caravan", new[] { "station", "bus_station", "ferry_terminal", "train_station" } },
            { "inn", new[] { "hotel", "hostel", "guest_house" } },
        };

        private sealed class Style { public Color Colour; public string Glyph; public string Word; }

        private static readonly Dictionary<string, Style> Styles = new Dictionary<string, Style>
        {
            { "hearth", new Style { Colour = Hex("D9822B"), Glyph = "H", Word = "hearth" } },
            { "vault", new Style { Colour = Hex("E9C46A"), Glyph = "V", Word = "vault" } },
            { "wardrobe", new Style { Colour = Hex("9B5DE5"), Glyph = "W", Word = "wardrobe" } },
            { "forge", new Style { Colour = Hex("C8553D"), Glyph = "F", Word = "forge" } },
            { "apothecary", new Style { Colour = Hex("2A9D8F"), Glyph = "A", Word = "apothecary" } },
            { "scriptorium", new Style { Colour = Hex("4A7BD0"), Glyph = "S", Word = "scriptorium" } },
            { "beacon", new Style { Colour = Hex("F4E9C8"), Glyph = "B", Word = "beacon" } },
            { "spring", new Style { Colour = Hex("5BC0EB"), Glyph = "~", Word = "spring" } },
            { "caravan", new Style { Colour = Hex("8D99AE"), Glyph = "C", Word = "caravan" } },
            { "inn", new Style { Colour = Hex("B5651D"), Glyph = "I", Word = "inn" } },
        };

        private static readonly Color Ink = new Color(0.37f, 0.20f, 0.20f);
        private static readonly Color Wall = new Color(0.86f, 0.78f, 0.60f);

        private readonly Transform _templates;
        private readonly Dictionary<string, GameObject> _built = new Dictionary<string, GameObject>();
        public int Placed { get; private set; }

        public ServiceMarkers()
        {
            _templates = new GameObject("ServiceTemplates").transform; // each finished template is deactivated; clones are activated on placement
        }

        /// <summary>The inactive template the map clones for a service.</summary>
        public GameObject Template(string service)
        {
            if (_built.TryGetValue(service, out var go)) return go;
            var style = Styles.TryGetValue(service, out var s) ? s : new Style { Colour = Color.gray, Glyph = "?", Word = service };

            go = new GameObject("service " + service);
            go.transform.SetParent(_templates, false);

            // An ink building: parchment walls, a coloured roof block, a dark plinth line.
            var plinth = Block(go.transform, "Plinth", new Vector3(12f, 0.3f, 12f), new Vector3(0, 0.15f, 0), Ink);
            var walls = Block(go.transform, "Walls", new Vector3(9f, 7f, 9f), new Vector3(0, 3.8f, 0), Wall);
            var roof = Block(go.transform, "Roof", new Vector3(10.5f, 2f, 10.5f), new Vector3(0, 8.3f, 0), style.Colour);
            roof.transform.localRotation = Quaternion.Euler(0, 45f, 0);
            var cap = Block(go.transform, "Cap", new Vector3(6.5f, 1.8f, 6.5f), new Vector3(0, 10.1f, 0), style.Colour);
            cap.transform.localRotation = Quaternion.Euler(0, 45f, 0);
            // A tall ink post carrying the sign, so the word reads over neighbouring outlines.
            Block(go.transform, "Post", new Vector3(0.5f, 10f, 0.5f), new Vector3(0, 15f, 0), Ink);
            foreach (var c in go.GetComponentsInChildren<Collider>(true)) Object.Destroy(c); // no tap target on services in phase 0

            // The sign: glyph and function word, facing the camera.
            var signGo = new GameObject("Sign");
            signGo.transform.SetParent(go.transform, false);
            signGo.transform.localPosition = new Vector3(0, 24f, 0);
            signGo.AddComponent<FaceCamera>();
            var sign = signGo.AddComponent<TextMeshPro>();
            sign.text = $"<b>{style.Glyph}</b>\n<size=55%>{style.Word}</size>";
            sign.fontSize = 26;
            sign.fontStyle = FontStyles.Bold;
            sign.alignment = TextAlignmentOptions.Center;
            sign.color = style.Colour;
            sign.outlineWidth = 0.3f;
            sign.outlineColor = Ink;
            sign.rectTransform.sizeDelta = new Vector2(90, 30);

            go.SetActive(false); // built active so TextMeshPro has a material; the template itself never renders
            _built[service] = go;
            return go;
        }

        /// <summary>Called by the map for every placed clone.</summary>
        public void OnPoi(string service, string poiName, GameObject clone)
        {
            clone.SetActive(true);
            Placed++;
            var sign = clone.GetComponentInChildren<TextMeshPro>(true);
            if (sign != null && !string.IsNullOrEmpty(poiName))
                sign.text += $"\n<size=40%><color=#5E3434>{poiName}</color></size>";
        }

        private static GameObject Block(Transform parent, string name, Vector3 size, Vector3 at, Color colour)
        {
            var b = GameObject.CreatePrimitive(PrimitiveType.Cube);
            b.name = name;
            b.transform.SetParent(parent, false);
            b.transform.localScale = size;
            b.transform.localPosition = at;
            var r = b.GetComponent<Renderer>();
            bool srp = UnityEngine.Rendering.GraphicsSettings.currentRenderPipeline != null;
            var shader = (srp ? Shader.Find("Universal Render Pipeline/Unlit") : null) ?? Shader.Find("Unlit/Color") ?? Shader.Find("Standard");
            r.sharedMaterial = new Material(shader) { color = colour };
            r.shadowCastingMode = UnityEngine.Rendering.ShadowCastingMode.Off;
            r.receiveShadows = false;
            return b;
        }

        private static Color Hex(string h) { ColorUtility.TryParseHtmlString("#" + h, out var c); return c; }

        /// <summary>Turns the sign to face the main camera around the vertical axis.</summary>
        public sealed class FaceCamera : MonoBehaviour
        {
            private void LateUpdate()
            {
                var cam = Camera.main;
                if (cam == null) return;
                var fwd = transform.position - cam.transform.position;
                fwd.y = 0;
                if (fwd.sqrMagnitude > 0.001f) transform.rotation = Quaternion.LookRotation(fwd);
            }
        }
    }
}
