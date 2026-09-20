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
        /// <summary>For full-screen views, so the HUD underneath never shows through.</summary>
        public static readonly Color PanelOpaque = new Color(0.08f, 0.09f, 0.11f, 1f);
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
            t.textWrappingMode = TextWrappingModes.Normal;
            t.raycastTarget = false;
            var le = go.AddComponent<LayoutElement>();
            le.minHeight = size * 1.2f;
            return t;
        }

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
            var t = Text(go.GetComponent<RectTransform>(), label, 44, false, align, textColour ?? Color.black);
            var trt = t.GetComponent<RectTransform>();
            trt.anchorMin = Vector2.zero;
            trt.anchorMax = Vector2.one;
            trt.offsetMin = new Vector2(24, 8);
            trt.offsetMax = new Vector2(-24, -8);
            return b;
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

        /// <summary>A title line with a close control, for full-screen views.</summary>
        public void Header(RectTransform parent, string title, System.Action close)
        {
            var row = Row(parent, 90, 16);
            var t = Text(row, title, 56, false, TextAlignmentOptions.Left, Accent);
            t.GetComponent<LayoutElement>().flexibleWidth = 1;
            var b = Button_(row, "Close", PanelLight, close, 90, TextAlignmentOptions.Center, Ink);
            var le = b.GetComponent<LayoutElement>();
            le.flexibleWidth = 0;
            le.minWidth = le.preferredWidth = 220;
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
