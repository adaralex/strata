using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace Strata.UI
{
    /// <summary>
    /// Builds the whole HUD from code so the scene needs no hand-wired Canvas. Two fonts:
    /// the default sans for everything the game says, and a serif for facts, so fact and
    /// fiction never share a typeface (non-negotiable 5).
    /// </summary>
    public sealed class UIKit
    {
        public readonly Canvas Canvas;
        public readonly RectTransform Root;
        public readonly TMP_FontAsset Sans;
        public readonly TMP_FontAsset Serif;

        public static readonly Color Ink = new Color(0.95f, 0.94f, 0.90f);
        public static readonly Color Panel = new Color(0.08f, 0.09f, 0.11f, 0.92f);
        public static readonly Color PanelLight = new Color(0.16f, 0.17f, 0.20f, 0.95f);
        public static readonly Color Warn = new Color(0.85f, 0.35f, 0.25f, 0.95f);
        public static readonly Color Accent = new Color(0.93f, 0.77f, 0.42f);

        public UIKit()
        {
            var go = new GameObject("StrataHUD", typeof(Canvas), typeof(CanvasScaler), typeof(GraphicRaycaster));
            Canvas = go.GetComponent<Canvas>();
            Canvas.renderMode = RenderMode.ScreenSpaceOverlay;
            Canvas.sortingOrder = 100;
            var scaler = go.GetComponent<CanvasScaler>();
            scaler.uiScaleMode = CanvasScaler.ScaleMode.ScaleWithScreenSize;
            scaler.referenceResolution = new Vector2(1080, 2340);
            scaler.matchWidthOrHeight = 0.5f;
            Root = go.GetComponent<RectTransform>();
            EnsureEventSystem();

            Sans = TMP_Settings.defaultFontAsset;
            var serifFont = Resources.Load<Font>("Fonts/Spectral-Regular");
            Serif = serifFont != null ? TMP_FontAsset.CreateFontAsset(serifFont) : Sans;
        }

        private static void EnsureEventSystem()
        {
            if (Object.FindFirstObjectByType<UnityEngine.EventSystems.EventSystem>() != null) return;
            var es = new GameObject("EventSystem", typeof(UnityEngine.EventSystems.EventSystem));
#if ENABLE_INPUT_SYSTEM && !ENABLE_LEGACY_INPUT_MANAGER
            es.AddComponent<UnityEngine.InputSystem.UI.InputSystemUIInputModule>();
#else
            es.AddComponent<UnityEngine.EventSystems.StandaloneInputModule>();
#endif
        }

        public RectTransform Panel_(string name, RectTransform parent, Color colour)
        {
            var go = new GameObject(name, typeof(RectTransform), typeof(Image));
            var rt = go.GetComponent<RectTransform>();
            rt.SetParent(parent, false);
            go.GetComponent<Image>().color = colour;
            return rt;
        }

        /// <summary>A panel anchored to an edge: top, bottom, or full screen.</summary>
        public RectTransform Bar(string name, RectTransform parent, Color colour, bool top, float height)
        {
            var rt = Panel_(name, parent, colour);
            rt.anchorMin = top ? new Vector2(0, 1) : new Vector2(0, 0);
            rt.anchorMax = top ? new Vector2(1, 1) : new Vector2(1, 0);
            rt.pivot = top ? new Vector2(0.5f, 1) : new Vector2(0.5f, 0);
            rt.anchoredPosition = Vector2.zero;
            rt.sizeDelta = new Vector2(0, height);
            return rt;
        }

        public RectTransform Full(string name, RectTransform parent, Color colour)
        {
            var rt = Panel_(name, parent, colour);
            rt.anchorMin = Vector2.zero;
            rt.anchorMax = Vector2.one;
            rt.offsetMin = rt.offsetMax = Vector2.zero;
            return rt;
        }

        public VerticalLayoutGroup Column(RectTransform rt, int padding = 32, float spacing = 16)
        {
            var v = rt.gameObject.AddComponent<VerticalLayoutGroup>();
            v.padding = new RectOffset(padding, padding, padding, padding);
            v.spacing = spacing;
            v.childForceExpandHeight = false;
            v.childControlHeight = true;
            v.childControlWidth = true;
            return v;
        }

        public TextMeshProUGUI Text(RectTransform parent, string text, float size, bool serif = false, TextAlignmentOptions align = TextAlignmentOptions.TopLeft, Color? colour = null)
        {
            var go = new GameObject("Text", typeof(RectTransform));
            go.transform.SetParent(parent, false);
            var t = go.AddComponent<TextMeshProUGUI>();
            t.font = serif ? Serif : Sans;
            t.text = text;
            t.fontSize = size;
            t.color = colour ?? Ink;
            t.alignment = align;
            t.enableWordWrapping = true;
            t.raycastTarget = false;
            var le = go.AddComponent<LayoutElement>();
            le.minHeight = size * 1.2f;
            return t;
        }

        public Button Button_(RectTransform parent, string label, Color colour, System.Action onClick, float height = 120)
        {
            var go = new GameObject("Button " + label, typeof(RectTransform), typeof(Image), typeof(Button));
            go.transform.SetParent(parent, false);
            go.GetComponent<Image>().color = colour;
            var b = go.GetComponent<Button>();
            b.onClick.AddListener(() => onClick());
            var le = go.AddComponent<LayoutElement>();
            le.minHeight = height;
            le.preferredHeight = height;
            var t = Text(go.GetComponent<RectTransform>(), label, 44, false, TextAlignmentOptions.Center, Color.black);
            var trt = t.GetComponent<RectTransform>();
            trt.anchorMin = Vector2.zero;
            trt.anchorMax = Vector2.one;
            trt.offsetMin = trt.offsetMax = Vector2.zero;
            return b;
        }

        public TMP_InputField Input(RectTransform parent, string value, string placeholder)
        {
            var go = new GameObject("Input", typeof(RectTransform), typeof(Image));
            go.transform.SetParent(parent, false);
            go.GetComponent<Image>().color = PanelLight;
            var le = go.AddComponent<LayoutElement>();
            le.minHeight = 110;
            var textArea = new GameObject("Text Area", typeof(RectTransform), typeof(RectMask2D));
            textArea.transform.SetParent(go.transform, false);
            var tart = textArea.GetComponent<RectTransform>();
            tart.anchorMin = Vector2.zero;
            tart.anchorMax = Vector2.one;
            tart.offsetMin = new Vector2(20, 10);
            tart.offsetMax = new Vector2(-20, -10);
            var text = Text(tart, "", 40);
            var ph = Text(tart, placeholder, 40, false, TextAlignmentOptions.Left, new Color(0.6f, 0.6f, 0.6f));
            foreach (var t in new[] { text, ph })
            {
                var rt = t.GetComponent<RectTransform>();
                rt.anchorMin = Vector2.zero;
                rt.anchorMax = Vector2.one;
                rt.offsetMin = rt.offsetMax = Vector2.zero;
                t.alignment = TextAlignmentOptions.Left;
            }
            var input = go.AddComponent<TMP_InputField>();
            input.textViewport = tart;
            input.textComponent = text;
            input.placeholder = ph;
            input.text = value;
            input.fontAsset = Sans;
            return input;
        }

        public Image Segment(RectTransform parent, Color colour, float weight)
        {
            var go = new GameObject("Segment", typeof(RectTransform), typeof(Image), typeof(LayoutElement));
            go.transform.SetParent(parent, false);
            go.GetComponent<Image>().color = colour;
            go.GetComponent<LayoutElement>().flexibleWidth = Mathf.Max(0.01f, weight);
            return go.GetComponent<Image>();
        }
    }
}
