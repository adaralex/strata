using System;
using System.Text;
using TMPro;
using UnityEngine;
using UnityEngine.UI;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The bar at the top: the civilizations under your feet as coloured segments, a legend
    /// in the same colours with their shares, then purity, the solar phase, the minutes left
    /// in the epoch and any wild or beacon flags. This is how a tester sees the ground change.
    /// </summary>
    public sealed class GroundStrip
    {
        public const float Height = 230;

        private readonly RectTransform _segments;
        private readonly TextMeshProUGUI _legend, _line2, _banner;
        private readonly RectTransform _bannerRt;
        private long _epochEnds;
        private string _line2Head = "", _line2Tail = "";

        public GroundStrip(UIKit ui)
        {
            var bar = ui.Bar("GroundStrip", ui.Root, UIKit.Panel, true, Height);
            var col = ui.Column(bar, 24, 10);
            col.childAlignment = TextAnchor.UpperLeft;

            var segGo = new GameObject("Segments", typeof(RectTransform), typeof(HorizontalLayoutGroup), typeof(LayoutElement));
            segGo.transform.SetParent(bar, false);
            _segments = segGo.GetComponent<RectTransform>();
            var h = segGo.GetComponent<HorizontalLayoutGroup>();
            h.spacing = 3;
            h.childForceExpandWidth = true;
            h.childControlWidth = true;
            segGo.GetComponent<LayoutElement>().minHeight = 28;
            ui.Frame(_segments, UIKit.GoldDim, 1.5f, -3); // a gold rule just outside the segments

            _legend = ui.Text(bar, "waiting for the ground", 36);
            _line2 = ui.Text(bar, "", 32, false, TextAlignmentOptions.TopLeft, new Color(0.72f, 0.72f, 0.72f));

            _bannerRt = ui.Bar("Banner", ui.Root, UIKit.Warn, true, 90);
            _bannerRt.anchoredPosition = new Vector2(0, -Height);
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
                _legend.text = "outside the world snapshot";
                _line2Head = _line2Tail = "";
                _line2.text = "";
                return;
            }
            var legend = new StringBuilder();
            foreach (var w in lk.Weights)
            {
                ui.Segment(_segments, CivPalette.Of(w.Civ), (float)w.Weight);
                legend.Append(CivPalette.Coloured(w.Civ)).Append(' ').Append((w.Weight * 100).ToString("0")).Append("%   ");
            }
            if (lk.Residual > 0.005)
            {
                ui.Segment(_segments, CivPalette.Others, (float)lk.Residual);
                legend.Append($"<color=#{ColorUtility.ToHtmlStringRGB(CivPalette.Others)}>others</color> {lk.Residual * 100:0}%");
            }
            _legend.text = legend.ToString().TrimEnd();

            var cond = lk.Cond;
            _line2Head = $"purity {lk.Purity * 100:0}%  ·  {cond?.Phase ?? "?"}";
            var tail = new StringBuilder();
            if (lk.Wild) tail.Append("  ·  wild");
            if (lk.BeaconGrade > 0) tail.Append($"  ·  beacon g{lk.BeaconGrade}");
            if (lk.Beacons.Count > 0) tail.Append($"  ·  in {lk.Beacons[0].Name}");
            if (lk.Excluded) tail.Append("  ·  no interaction here");
            _line2Tail = tail.ToString();
            Tick();
        }

        public void SetEpochEnds(long unix) => _epochEnds = unix;

        /// <summary>Refresh the countdown only; the rest waits for the next lookup.</summary>
        public void Tick()
        {
            if (_line2 == null || _line2Head.Length == 0) return;
            var epoch = EpochLeft();
            _line2.text = _line2Head + (epoch.Length > 0 ? "  ·  " + epoch : "") + _line2Tail;
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
