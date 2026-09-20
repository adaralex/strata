using System;
using System.Text;
using TMPro;
using UnityEngine;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The drop, as the server returned it. Fact and fiction live in separate panels with
    /// separate typefaces and a rule between them (non-negotiable 5). A hybrid shows no
    /// fact panel at all and says so.
    /// </summary>
    public sealed class ItemCard
    {
        private readonly UIKit _ui;

        public ItemCard(UIKit ui) { _ui = ui; }

        public void Show(CollapseResult res, Action closed)
        {
            var item = res.Item;
            var panel = _ui.Full("ItemCard", _ui.Root, UIKit.Panel);
            var col = _ui.Column(panel, 40, 20);
            col.childAlignment = TextAnchor.UpperLeft;

            _ui.Text(panel, $"<color=#{ColorUtility.ToHtmlStringRGB(CivPalette.Of(item.Civ))}>{CivPalette.Label(item.Civ)}</color>  ·  {item.TierName}  ·  {item.Authenticity}", 38);
            _ui.Text(panel, item.Name, 64, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            if (item.IsHybrid)
                _ui.Text(panel, $"a hybrid: {CivPalette.Label(item.Civ)} carried on {CivPalette.Label(item.Secondary)} ground", 36, false, TextAlignmentOptions.TopLeft, new Color(0.8f, 0.8f, 0.8f));

            var sb = new StringBuilder();
            if (item.Kind == "edible" && item.Edible?.Buff != null)
                sb.Append($"edible: +{item.Edible.Buff.Pct:0}% {item.Edible.Buff.Stat} for {item.Edible.Buff.Minutes} min");
            else if (item.Affixes != null)
                foreach (var a in item.Affixes) sb.Append($"{a.K} +{a.V:0}   ");
            if (sb.Length > 0) _ui.Text(panel, sb.ToString().TrimEnd(), 38);

            var prov = item.Provenance;
            var where = prov?.Kind switch
            {
                "museum" => $"accessioned at {prov.Name}",
                "site" => $"taken at the site of {prov.Name}",
                _ => $"from the soil of cell {prov?.Cell}",
            };
            if (res.Beacon != null && prov?.Kind == "soil") where += $", within {res.Beacon.Name}";
            _ui.Text(panel, where, 34, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            if (!string.IsNullOrEmpty(item.CodexRef))
                _ui.Text(panel, $"a codex fragment came with it: {item.CodexRef}", 34, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));

            // Fact panel: serif, light. Absent for hybrids.
            if (!item.IsHybrid && !string.IsNullOrEmpty(item.Fact))
            {
                var fact = _ui.Panel_("Fact", panel, new Color(0.93f, 0.90f, 0.82f));
                _ui.Column(fact, 28, 8);
                _ui.Text(fact, "WHAT THIS WAS", 26, true, TextAlignmentOptions.TopLeft, new Color(0.35f, 0.3f, 0.25f));
                _ui.Text(fact, item.Fact, 38, true, TextAlignmentOptions.TopLeft, new Color(0.12f, 0.1f, 0.08f));
            }
            else if (item.IsHybrid)
            {
                var note = _ui.Panel_("NoFact", panel, UIKit.PanelLight);
                _ui.Column(note, 28, 8);
                _ui.Text(note, "This object was made by the transmission and never existed above ground. There is nothing true to say about it.", 34, false, TextAlignmentOptions.TopLeft, new Color(0.75f, 0.75f, 0.75f));
            }

            // Fiction panel: sans, dark.
            if (!string.IsNullOrEmpty(item.Fiction))
            {
                var fic = _ui.Panel_("Fiction", panel, UIKit.PanelLight);
                _ui.Column(fic, 28, 8);
                _ui.Text(fic, "THE ANVIL SAYS", 26, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
                _ui.Text(fic, item.Fiction, 38, false, TextAlignmentOptions.TopLeft);
            }

            _ui.Button_(panel, "Keep walking", UIKit.Accent, () =>
            {
                UnityEngine.Object.Destroy(panel.gameObject);
                closed?.Invoke();
            });
        }
    }
}
