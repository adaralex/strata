using TMPro;
using UnityEngine;
using UnityEngine.UI;

namespace Strata.UI
{
    /// <summary>
    /// Builds the whole HUD from code so the scene needs no hand-wired Canvas. Three type
    /// styles with strict jobs: the default sans for everything the game says, Spectral
    /// (serif) for facts only, so fact and fiction never share a typeface (non-negotiable 5),
    /// and Cinzel (display, capitals) for chrome: titles, buttons, item and monster names.
    /// The look is leather and gold over the parchment map; every frame is drawn, not art.
    /// </summary>
    public sealed class UIKit
    {
        public readonly Canvas Canvas;
        public readonly RectTransform Root;
        public readonly TMP_FontAsset Sans;
        public readonly TMP_FontAsset Serif;
        public readonly TMP_FontAsset Display;

        public static readonly Color Ink = new Color(0.93f, 0.87f, 0.72f);            // parchment text on leather
        public static readonly Color InkDark = new Color(0.13f, 0.09f, 0.07f);         // text on gold
        public static readonly Color Muted = new Color(0.66f, 0.60f, 0.50f);
        public static readonly Color Panel = new Color(0.13f, 0.09f, 0.07f, 0.94f);   // leather
        public static readonly Color PanelOpaque = new Color(0.13f, 0.09f, 0.07f, 1f);
        public static readonly Color PanelLight = new Color(0.22f, 0.16f, 0.11f, 0.98f);
        public static readonly Color Warn = new Color(0.62f, 0.20f, 0.16f, 0.96f);
        public static readonly Color Accent = new Color(0.85f, 0.68f, 0.35f);         // gold
        public static readonly Color GoldDim = new Color(0.55f, 0.43f, 0.22f);

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
            Serif = Runtime("Fonts/Spectral-Regular") ?? Sans;
            Display = Runtime("Fonts/Cinzel-Variable") ?? Sans;
        }

        private static TMP_FontAsset Runtime(string resource)
        {
            var font = Resources.Load<Font>(resource);
            if (font == null) { Debug.LogWarning("UIKit: font resource missing: " + resource); return null; }
            try { return TMP_FontAsset.CreateFontAsset(font); }
            catch (System.Exception e) { Debug.LogWarning("UIKit: could not create font asset for " + resource + ": " + e.Message); return null; }
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

        // ---- panels ----

        public RectTransform Panel_(string name, RectTransform parent, Color colour)
        {
            var go = new GameObject(name, typeof(RectTransform), typeof(Image));
            var rt = go.GetComponent<RectTransform>();
            rt.SetParent(parent, false);
            go.GetComponent<Image>().color = colour;
            return rt;
        }

        /// <summary>A panel with a thin gold frame inside its edge.</summary>
        public RectTransform Framed(string name, RectTransform parent, Color colour, float thickness = 1.5f)
        {
            var rt = Panel_(name, parent, colour);
            Frame(rt, GoldDim, thickness, 4);
            return rt;
        }

        /// <summary>A panel anchored to an edge: top, bottom, or full screen. Carries a gold hairline on its inner edge.</summary>
        public RectTransform Bar(string name, RectTransform parent, Color colour, bool top, float height)
        {
            var rt = Panel_(name, parent, colour);
            rt.anchorMin = top ? new Vector2(0, 1) : new Vector2(0, 0);
            rt.anchorMax = top ? new Vector2(1, 1) : new Vector2(1, 0);
            rt.pivot = top ? new Vector2(0.5f, 1) : new Vector2(0.5f, 0);
            rt.anchoredPosition = Vector2.zero;
            rt.sizeDelta = new Vector2(0, height);
            Edge(rt, top ? Vector2.zero : Vector2.up, GoldDim, 2f);
            return rt;
        }

        public RectTransform Full(string name, RectTransform parent, Color colour)
        {
            var rt = Panel_(name, parent, colour);
            rt.anchorMin = Vector2.zero;
            rt.anchorMax = Vector2.one;
            rt.offsetMin = rt.offsetMax = Vector2.zero;
            Frame(rt, GoldDim, 2f, 12);
            return rt;
        }

        /// <summary>Four thin lines inside a rect's edges, outside any layout group's flow.</summary>
        public void Frame(RectTransform rt, Color colour, float thickness, float inset)
        {
            Line(rt, "FrameTop", new Vector2(0, 1), new Vector2(1, 1), new Vector2(inset, -inset - thickness), new Vector2(-inset, -inset), colour);
            Line(rt, "FrameBottom", new Vector2(0, 0), new Vector2(1, 0), new Vector2(inset, inset), new Vector2(-inset, inset + thickness), colour);
            Line(rt, "FrameLeft", new Vector2(0, 0), new Vector2(0, 1), new Vector2(inset, inset), new Vector2(inset + thickness, -inset), colour);
            Line(rt, "FrameRight", new Vector2(1, 0), new Vector2(1, 1), new Vector2(-inset - thickness, inset), new Vector2(-inset, -inset), colour);
        }

        /// <summary>One hairline along an edge: anchor (0,0) bottom or (0,1) top.</summary>
        private void Edge(RectTransform rt, Vector2 edge, Color colour, float thickness)
        {
            bool top = edge.y > 0.5f;
            Line(rt, top ? "EdgeTop" : "EdgeBottom", new Vector2(0, top ? 1 : 0), new Vector2(1, top ? 1 : 0),
                new Vector2(0, top ? -thickness : 0), new Vector2(0, top ? 0 : thickness), colour);
        }

        private static void Line(RectTransform parent, string name, Vector2 anchorMin, Vector2 anchorMax, Vector2 offsetMin, Vector2 offsetMax, Color colour)
        {
            var go = new GameObject(name, typeof(RectTransform), typeof(Image), typeof(LayoutElement));
            go.transform.SetParent(parent, false);
            var img = go.GetComponent<Image>();
            img.color = colour;
            img.raycastTarget = false;
            go.GetComponent<LayoutElement>().ignoreLayout = true;
            var rt = go.GetComponent<RectTransform>();
            rt.anchorMin = anchorMin;
            rt.anchorMax = anchorMax;
            rt.offsetMin = offsetMin;
            rt.offsetMax = offsetMax;
        }

        // ---- layout ----

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

        /// <summary>A horizontal strip of equal-width children inside a column.</summary>
        public RectTransform Row(RectTransform parent, float height, float spacing = 16)
        {
            var go = new GameObject("Row", typeof(RectTransform), typeof(HorizontalLayoutGroup), typeof(LayoutElement));
            go.transform.SetParent(parent, false);
            var h = go.GetComponent<HorizontalLayoutGroup>();
            h.spacing = spacing;
            h.childForceExpandWidth = true;
            h.childControlWidth = true;
            h.childControlHeight = true;
            h.childForceExpandHeight = true;
            var le = go.GetComponent<LayoutElement>();
            le.minHeight = height;
            le.preferredHeight = height;
            le.flexibleHeight = 0; // the group would otherwise report its children's flexible height and swell
            return go.GetComponent<RectTransform>();
        }

        /// <summary>A vertical scroll area that takes the column's remaining height; returns its content column.</summary>
        public RectTransform Scroll(RectTransform parent, float spacing = 16)
        {
            var go = new GameObject("Scroll", typeof(RectTransform), typeof(ScrollRect), typeof(LayoutElement));
            go.transform.SetParent(parent, false);
            var le = go.GetComponent<LayoutElement>();
            le.flexibleHeight = 1;
            le.minHeight = 200;

            var viewport = new GameObject("Viewport", typeof(RectTransform), typeof(RectMask2D), typeof(Image));
            viewport.transform.SetParent(go.transform, false);
            var vimg = viewport.GetComponent<Image>();
            vimg.color = new Color(0, 0, 0, 0.001f);
            vimg.raycastTarget = true; // dragging needs a target
            var vrt = viewport.GetComponent<RectTransform>();
            vrt.anchorMin = Vector2.zero;
            vrt.anchorMax = Vector2.one;
            vrt.offsetMin = vrt.offsetMax = Vector2.zero;

            var content = new GameObject("Content", typeof(RectTransform), typeof(VerticalLayoutGroup), typeof(ContentSizeFitter));
            content.transform.SetParent(viewport.transform, false);
            var crt = content.GetComponent<RectTransform>();
            crt.anchorMin = new Vector2(0, 1);
            crt.anchorMax = new Vector2(1, 1);
            crt.pivot = new Vector2(0.5f, 1);
            crt.offsetMin = crt.offsetMax = Vector2.zero;
            var v = content.GetComponent<VerticalLayoutGroup>();
            v.padding = new RectOffset(0, 0, 0, 40);
            v.spacing = spacing;
            v.childForceExpandHeight = false;
            v.childControlHeight = true;
            v.childControlWidth = true;
            content.GetComponent<ContentSizeFitter>().verticalFit = ContentSizeFitter.FitMode.PreferredSize;

            var sr = go.GetComponent<ScrollRect>();
            sr.content = crt;
            sr.viewport = vrt;
            sr.horizontal = false;
            sr.vertical = true;
            sr.movementType = ScrollRect.MovementType.Clamped;
            sr.scrollSensitivity = 40;
            return crt;
        }

        // ---- text ----

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
            t.textWrappingMode = TextWrappingModes.Normal;
            t.raycastTarget = false;
            var le = go.AddComponent<LayoutElement>();
            le.minHeight = size * 1.2f;
            return t;
        }

        /// <summary>Chrome text in the display face: titles, section heads, item and monster names.</summary>
        public TextMeshProUGUI Title(RectTransform parent, string text, float size, TextAlignmentOptions align = TextAlignmentOptions.TopLeft, Color? colour = null)
        {
            var t = Text(parent, text, size, false, align, colour ?? Accent);
            t.font = Display;
            t.characterSpacing = 4;
            return t;
        }

        /// <summary>A title line with a close control, for full-screen views.</summary>
        public void Header(RectTransform parent, string title, System.Action close)
        {
            var row = Row(parent, 90, 16);
            var t = Title(row, title, 52, TextAlignmentOptions.Left);
            t.GetComponent<LayoutElement>().flexibleWidth = 1;
            var b = Button_(row, "Close", PanelLight, close, 90);
            var le = b.GetComponent<LayoutElement>();
            le.flexibleWidth = 0;
            le.minWidth = le.preferredWidth = 220;
        }

        // ---- controls ----

        /// <summary>
        /// A button. Gold (Accent) buttons are primary: gold fill, dark display text. Any other
        /// colour draws a framed leather button with gold display text. Multi-line labels are
        /// rows (lists), set in the sans so they stay readable.
        /// </summary>
        public Button Button_(RectTransform parent, string label, Color colour, System.Action onClick, float height = 120, TextAlignmentOptions align = TextAlignmentOptions.Center, Color? textColour = null)
        {
            var go = new GameObject("Button " + label, typeof(RectTransform), typeof(Image), typeof(Button));
            go.transform.SetParent(parent, false);
            go.GetComponent<Image>().color = colour;
            var b = go.GetComponent<Button>();
            b.onClick.AddListener(() => onClick());
            var le = go.AddComponent<LayoutElement>();
            le.minHeight = height;
            le.preferredHeight = height;
            bool primary = colour == Accent;
            bool row = label.Contains("\n");
            var rt = go.GetComponent<RectTransform>();
            Frame(rt, primary ? InkDark : GoldDim, primary ? 1.5f : 1.5f, 5);
            var t = Text(rt, label, row ? 40 : 38, false, align, textColour ?? (primary ? InkDark : Accent));
            if (!row) { t.font = Display; t.characterSpacing = 3; }
            var trt = t.GetComponent<RectTransform>();
            trt.anchorMin = Vector2.zero;
            trt.anchorMax = Vector2.one;
            trt.offsetMin = new Vector2(28, 8);
            trt.offsetMax = new Vector2(-28, -8);
            return b;
        }

        public TMP_InputField Input(RectTransform parent, string value, string placeholder)
        {
            var go = new GameObject("Input", typeof(RectTransform), typeof(Image));
            go.transform.SetParent(parent, false);
            go.GetComponent<Image>().color = PanelLight;
            var le = go.AddComponent<LayoutElement>();
            le.minHeight = 110;
            Frame(go.GetComponent<RectTransform>(), GoldDim, 1.5f, 4);
            var textArea = new GameObject("Text Area", typeof(RectTransform), typeof(RectMask2D));
            textArea.transform.SetParent(go.transform, false);
            var tart = textArea.GetComponent<RectTransform>();
            tart.anchorMin = Vector2.zero;
            tart.anchorMax = Vector2.one;
            tart.offsetMin = new Vector2(20, 10);
            tart.offsetMax = new Vector2(-20, -10);
            var text = Text(tart, "", 40);
            var ph = Text(tart, placeholder, 40, false, TextAlignmentOptions.Left, Muted);
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
