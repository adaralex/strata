using System;
using UnityEngine;
using UnityEngine.EventSystems;

namespace Strata.UI
{
    /// <summary>
    /// Reports taps on the map. A tap is a press that ends within MaxTapSeconds having moved
    /// less than MaxTapPixels, and that did not start over a HUD element; drags and pinches
    /// are left to GO Map's camera controls. Deliberately not a UI raycast target: a
    /// full-screen Image would make every touch count as "over UI", which is exactly the test
    /// GO Map's orbit uses to ignore a touch, and the map could no longer be rotated.
    /// Active Input Handling in this project is the Input Manager; the Input System branch
    /// is kept so the class still compiles and works if that changes.
    /// </summary>
    public sealed class TapCatcher : MonoBehaviour
    {
        public event Action<Vector2> OnTap;

        public float MaxTapSeconds = 0.35f;
        public float MaxTapPixels = 24f;

        private bool _down, _overUiAtStart;
        private Vector2 _start;
        private float _startTime;

        public static TapCatcher Create(RectTransform root)
        {
            var go = new GameObject("TapCatcher");
            go.transform.SetParent(root, false);
            return go.AddComponent<TapCatcher>();
        }

        private void Update()
        {
            if (TryRead(out var pressed, out var released, out var pos, out var fingerId))
            {
                if (pressed)
                {
                    _down = true;
                    _start = pos;
                    _startTime = Time.unscaledTime;
                    _overUiAtStart = IsOverUi(fingerId);
                }
                else if (released && _down)
                {
                    _down = false;
                    bool quick = Time.unscaledTime - _startTime <= MaxTapSeconds;
                    bool still = (pos - _start).sqrMagnitude <= MaxTapPixels * MaxTapPixels;
                    if (quick && still && !_overUiAtStart) OnTap?.Invoke(pos);
                }
            }
        }

        /// <summary>Same test GO Map uses, so a tap on the HUD never reaches the map.</summary>
        private static bool IsOverUi(int fingerId)
        {
            var es = EventSystem.current;
            if (es == null) return false;
            return fingerId >= 0 ? es.IsPointerOverGameObject(fingerId) : es.IsPointerOverGameObject();
        }

        // Reads the primary pointer: one touch when present, otherwise the mouse.
        private static bool TryRead(out bool pressed, out bool released, out Vector2 pos, out int fingerId)
        {
            pressed = released = false; pos = default; fingerId = -1;
#if ENABLE_INPUT_SYSTEM && !ENABLE_LEGACY_INPUT_MANAGER
            var ts = UnityEngine.InputSystem.Touchscreen.current;
            if (ts != null && ts.touches.Count > 0 && ts.primaryTouch.press.isPressed | ts.primaryTouch.press.wasReleasedThisFrame)
            {
                var t = ts.primaryTouch;
                pressed = t.press.wasPressedThisFrame;
                released = t.press.wasReleasedThisFrame;
                pos = t.position.ReadValue();
                fingerId = t.touchId.ReadValue();
                return pressed || released;
            }
            var mouse = UnityEngine.InputSystem.Mouse.current;
            if (mouse == null) return false;
            pressed = mouse.leftButton.wasPressedThisFrame;
            released = mouse.leftButton.wasReleasedThisFrame;
            pos = mouse.position.ReadValue();
            return pressed || released;
#else
            if (Input.touchCount > 0)
            {
                if (Input.touchCount > 1) { return false; } // pinch: never a tap
                var t = Input.GetTouch(0);
                pressed = t.phase == TouchPhase.Began;
                released = t.phase == TouchPhase.Ended;
                pos = t.position;
                fingerId = t.fingerId;
                return pressed || released;
            }
            pressed = Input.GetMouseButtonDown(0);
            released = Input.GetMouseButtonUp(0);
            pos = Input.mousePosition;
            return pressed || released;
#endif
        }

        /// <summary>Test hook: behaves as a completed tap at a screen position.</summary>
        public void Simulate(Vector2 screen) => OnTap?.Invoke(screen);
    }
}
