using System;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The placeholder fight: PLAN §8's shape with everything stripped. Three beats. A ring
    /// shrinks toward a target circle; tap while it is inside the window to land a hit;
    /// three hits collapse the monster, three misses and it slips away. Auto-resolve is
    /// always on screen (non-negotiable 6) and simply wins. Nothing here is valuable: the
    /// server decides the drop when the collapse is posted.
    /// </summary>
    public sealed class Fight
    {
        public const int BeatsToWin = 3;
        public const int MissesToLose = 3;
        public const float BeatSeconds = 1.6f;
        public const float WindowLow = 0.85f, WindowHigh = 1.15f;

        private readonly UIKit _ui;
        private RectTransform _panel;
        private RectTransform _ring, _target;
        private TextMeshProUGUI _title, _status;
        private Action<bool, bool> _done; // (won, autoResolved)
        private float _beatT;
        private int _hits, _misses;
        private bool _active;

        public bool Active => _active;

        public Fight(UIKit ui) { _ui = ui; }

        public void Begin(Spawn sp, Action<bool, bool> done)
        {
            _done = done;
            _hits = _misses = 0;
            _beatT = 0;
            _active = true;
            _panel = _ui.Full("Fight", _ui.Root, UIKit.Panel);
            var col = _ui.Column(_panel, 40, 24);
            col.childAlignment = TextAnchor.UpperCenter;
            _title = _ui.Text(_panel, $"{sp.Name}\n<size=60%>{CivPalette.Label(sp.Civ)} · {sp.TierName} · {sp.Rank}</size>", 60, false, TextAlignmentOptions.Center);
            _ui.Text(_panel, sp.Fiction ?? "", 38, false, TextAlignmentOptions.Center, new Color(0.8f, 0.8f, 0.8f));
            _status = _ui.Text(_panel, "tap when the ring meets the circle", 40, false, TextAlignmentOptions.Center, UIKit.Accent);

            var arena = new GameObject("Arena", typeof(RectTransform), typeof(LayoutElement));
            arena.transform.SetParent(_panel, false);
            arena.GetComponent<LayoutElement>().minHeight = 900;
            var art = arena.GetComponent<RectTransform>();
            _target = Circle(art, new Color(1, 1, 1, 0.25f), 400);
            _ring = Circle(art, CivPalette.Of(sp.Civ), 400);

            var tapZone = arena.AddComponent<Button>();
            tapZone.transition = Selectable.Transition.None;
            arena.AddComponent<Image>().color = new Color(0, 0, 0, 0.01f);
            tapZone.onClick.AddListener(Tap);

            _ui.Button_(_panel, "Auto-resolve", UIKit.Accent, () => Finish(true, true));
            _ui.Button_(_panel, "Walk away", UIKit.PanelLight, () => Finish(false, false), 90);
        }

        private RectTransform Circle(RectTransform parent, Color colour, float size)
        {
            var go = new GameObject("Circle", typeof(RectTransform), typeof(Image));
            go.transform.SetParent(parent, false);
            var img = go.GetComponent<Image>();
            img.color = colour;
            img.sprite = CircleSprite();
            img.raycastTarget = false;
            var rt = go.GetComponent<RectTransform>();
            rt.anchorMin = rt.anchorMax = new Vector2(0.5f, 0.5f);
            rt.sizeDelta = new Vector2(size, size);
            return rt;
        }

        private static Sprite _circle;
        private static Sprite CircleSprite()
        {
            if (_circle != null) return _circle;
            const int n = 128;
            var tex = new Texture2D(n, n, TextureFormat.RGBA32, false);
            var px = new Color[n * n];
            for (int y = 0; y < n; y++)
                for (int x = 0; x < n; x++)
                {
                    float dx = x - n / 2f + 0.5f, dy = y - n / 2f + 0.5f;
                    float d = Mathf.Sqrt(dx * dx + dy * dy) / (n / 2f);
                    px[y * n + x] = new Color(1, 1, 1, d > 0.82f && d < 1f ? 1f : 0f);
                }
            tex.SetPixels(px);
            tex.Apply();
            _circle = Sprite.Create(tex, new Rect(0, 0, n, n), new Vector2(0.5f, 0.5f));
            return _circle;
        }

        public void Update(float dt)
        {
            if (!_active || _ring == null) return;
            _beatT += dt;
            if (_beatT >= BeatSeconds)
            {
                _beatT -= BeatSeconds;
                Miss("too slow");
                if (!_active) return;
            }
            // Ring scale falls from 2.0 to 0.5 over the beat; the target is 1.0.
            float s = Mathf.Lerp(2.0f, 0.5f, _beatT / BeatSeconds);
            _ring.localScale = Vector3.one * s;
        }

        private void Tap()
        {
            if (!_active) return;
            float s = _ring.localScale.x;
            if (s >= WindowLow && s <= WindowHigh)
            {
                _hits++;
                _status.text = $"hit {_hits}/{BeatsToWin}";
                _status.color = UIKit.Accent;
                if (_hits >= BeatsToWin) { Finish(true, false); return; }
            }
            else
            {
                Miss(s > WindowHigh ? "early" : "late");
                if (!_active) return;
            }
            _beatT = 0;
        }

        private void Miss(string why)
        {
            _misses++;
            _status.text = $"{why}  ({_misses}/{MissesToLose})";
            _status.color = UIKit.Warn;
            if (_misses >= MissesToLose) Finish(false, false);
        }

        private void Finish(bool won, bool auto)
        {
            if (!_active) return;
            _active = false;
            UnityEngine.Object.Destroy(_panel.gameObject);
            _done?.Invoke(won, auto);
        }
    }
}
