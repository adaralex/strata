using System;
using UnityEngine;
using UnityEngine.EventSystems;
using UnityEngine.UI;

namespace Strata.UI
{
    /// <summary>
    /// A transparent full-screen image at the back of the HUD that reports taps through
    /// the UI event system. Works whichever input backend the project uses (legacy Input
    /// Manager or the Input System package); reading UnityEngine.Input directly throws
    /// under the new one, which is how the first build lost every tap.
    /// </summary>
    public sealed class TapCatcher : MonoBehaviour, IPointerClickHandler
    {
        public event Action<Vector2> OnTap;

        public static TapCatcher Create(RectTransform root)
        {
            var go = new GameObject("TapCatcher", typeof(RectTransform), typeof(Image));
            go.transform.SetParent(root, false);
            go.transform.SetAsFirstSibling(); // behind every panel, so panels win
            var rt = go.GetComponent<RectTransform>();
            rt.anchorMin = Vector2.zero;
            rt.anchorMax = Vector2.one;
            rt.offsetMin = rt.offsetMax = Vector2.zero;
            var img = go.GetComponent<Image>();
            img.color = new Color(0, 0, 0, 0.001f);
            img.raycastTarget = true;
            return go.AddComponent<TapCatcher>();
        }

        public void OnPointerClick(PointerEventData e) => OnTap?.Invoke(e.position);
    }
}
