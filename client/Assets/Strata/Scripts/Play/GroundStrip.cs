using System;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The bar at the top: the top four civilizations as coloured segments, purity, the
    /// solar phase and the minutes left in the epoch. This is how a tester sees the ground
    /// change under them.
    /// </summary>
    public sealed class GroundStrip
    {
        private readonly RectTransform _segments;
        private readonly TextMeshProUGUI _line1, _line2, _banner;
        private readonly RectTransform _bannerRt;
        private long _epochEnds;

        public GroundStrip(UIKit ui)
        {
            var bar = ui.Bar("GroundStrip", ui.Root, UIKit.Panel, true, 260);
            var col = ui.Column(bar, 24, 8);
            col.childAlignment = TextAnchor.UpperLeft;

            var segGo = new GameObject("Segments", typeof(RectTransform), typeof(HorizontalLayoutGroup), typeof(LayoutElement));
            segGo.transform.SetParent(bar, false);
            _segments = segGo.GetComponent<RectTransform>();
            var h = segGo.GetComponent<HorizontalLayoutGroup>();
            h.spacing = 4;
            h.childForceExpandWidth = true;
            h.childControlWidth = true;
            segGo.GetComponent<LayoutElement>().minHeight = 36;

            _line1 = ui.Text(bar, "waiting for the ground", 40);
            _line2 = ui.Text(bar, "", 34, false, TextAlignmentOptions.TopLeft, new Color(0.75f, 0.75f, 0.75f));

            _bannerRt = ui.Bar("Banner", ui.Root, UIKit.Warn, true, 90);
            _bannerRt.anchoredPosition = new Vector2(0, -260);
            _banner = ui.Text(_bannerRt, "", 38, false, TextAlignmentOptions.Center);
            var brt = _banner.GetComponent<RectTransform>();
            brt.anchorMin = Vector2.zero;
            brt.anchorMax = Vector2.one;
            brt.offsetMin = brt.offsetMax = Vector2.zero;
            _bannerRt.gameObject.SetActive(false);
        }

        public void Show(LookupOut lk, UIKit ui)
        {
            foreach (Transform c in _segments) UnityEngine.Object.Destroy(c.gameObject);
            if (lk == null || !lk.Found)
            {
                _line1.text = "outside the world snapshot";
                _line2.text = "";
                return;
            }
            var parts = new System.Text.StringBuilder();
            foreach (var w in lk.Weights)
            {
                ui.Segment(_segments, CivPalette.Of(w.Civ), (float)w.Weight);
                parts.Append($"{CivPalette.Label(w.Civ)} {w.Weight:0.00}  ");
            }
            if (lk.Residual > 0.005)
            {
                ui.Segment(_segments, new Color(0.4f, 0.4f, 0.4f), (float)lk.Residual);
                parts.Append($"others {lk.Residual:0.00}");
            }
            _line1.text = parts.ToString().TrimEnd();
            var cond = lk.Cond;
            var extras = "";
            if (lk.Wild) extras += "  wild";
            if (lk.BeaconGrade > 0) extras += $"  beacon g{lk.BeaconGrade}";
            if (lk.Beacons.Count > 0) extras += $"  IN {lk.Beacons[0].Name}";
            _line2.text = $"purity {lk.Purity:0.00}   {cond?.Phase ?? "?"}  prop {lk.Propagation:0.00}   {EpochLeft()}{extras}";
        }

        public void SetEpochEnds(long unix) => _epochEnds = unix;

        public void Tick()
        {
            // Refresh the countdown only; the rest waits for the next lookup.
            if (_line2 != null && _epochEnds > 0 && _line2.text.Contains("prop"))
            {
                var i = _line2.text.IndexOf("   ", _line2.text.IndexOf("prop", StringComparison.Ordinal), StringComparison.Ordinal);
                if (i > 0)
                {
                    var j = _line2.text.IndexOf("  ", i + 3, StringComparison.Ordinal);
                    var tail = j > 0 ? _line2.text.Substring(j) : "";
                    _line2.text = _line2.text.Substring(0, i + 3) + EpochLeft() + tail;
                }
            }
        }

        private string EpochLeft()
        {
            if (_epochEnds == 0) return "";
            var left = _epochEnds - DateTimeOffset.UtcNow.ToUnixTimeSeconds();
            if (left < 0) left = 0;
            return $"epoch {left / 60:0}:{left % 60:00}";
        }

        public void Banner(string text)
        {
            _bannerRt.gameObject.SetActive(!string.IsNullOrEmpty(text));
            _banner.text = text ?? "";
        }
    }
}
