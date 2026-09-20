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
    /// fact panel at all and says so. Describe() is shared with the inventory detail view.
    /// </summary>
    public sealed class ItemCard
    {
        private readonly UIKit _ui;

        public ItemCard(UIKit ui) { _ui = ui; }

        public void Show(CollapseResult res, Action closed)
        {
            var panel = _ui.Full("ItemCard", _ui.Root, UIKit.PanelOpaque);
            var col = _ui.Column(panel, 40, 20);
            col.childAlignment = TextAnchor.UpperLeft;
            var content = _ui.Scroll(panel);
            Describe(_ui, content, res.Item, res.Beacon);
            _ui.Button_(panel, "Keep walking", UIKit.Accent, () =>
            {
                UnityEngine.Object.Destroy(panel.gameObject);
                closed?.Invoke();
            });
        }

        /// <summary>Everything the card says about an item, minus the buttons.</summary>
        public static void Describe(UIKit ui, RectTransform panel, Item item, BeaconOut beacon)
        {
            ui.Text(panel, $"{CivPalette.Coloured(item.Civ)}  ·  {item.TierName}  ·  {item.Authenticity}{(string.IsNullOrEmpty(item.Slot) ? "" : "  ·  " + item.Slot)}", 38);
            ui.Text(panel, item.Name, 64, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            if (item.IsHybrid)
                ui.Text(panel, $"a hybrid: {CivPalette.Label(item.Civ)} carried on {CivPalette.Label(item.Secondary)} ground", 36, false, TextAlignmentOptions.TopLeft, new Color(0.8f, 0.8f, 0.8f));

            var sb = new StringBuilder();
            if (item.Kind == "edible" && item.Edible?.Buff != null)
                sb.Append($"edible: +{item.Edible.Buff.Pct:0}% {item.Edible.Buff.Stat} for {item.Edible.Buff.Minutes} min");
            else if (item.Affixes != null && item.Affixes.Count > 0)
                foreach (var a in item.Affixes) sb.Append($"{CivPalette.StatLabel(a.K)} +{a.V:0}   ");
            else if (item.Kind != "edible")
                sb.Append("no affixes at this tier");
            if (sb.Length > 0) ui.Text(panel, sb.ToString().TrimEnd(), 38);

            var prov = item.Provenance;
            var where = prov?.Kind switch
            {
                "museum" => $"accessioned at {prov.Name}",
                "site" => $"taken at the site of {prov.Name}",
                _ => $"from the soil of cell {prov?.Cell}",
            };
            if (beacon != null && prov?.Kind == "soil") where += $", within {beacon.Name}";
            if (!string.IsNullOrEmpty(prov?.AcquiredAt) && DateTimeOffset.TryParse(prov.AcquiredAt, out var at)) where += $"  ·  {at.ToLocalTime():d MMM HH:mm}";
            ui.Text(panel, where, 34, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            if (!string.IsNullOrEmpty(item.CodexRef))
                ui.Text(panel, $"a codex fragment came with it: {item.CodexRef}", 34, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));

            // Fact panel: serif, light. Absent for hybrids.
            if (!item.IsHybrid && !string.IsNullOrEmpty(item.Fact))
            {
                var fact = ui.Panel_("Fact", panel, new Color(0.93f, 0.90f, 0.82f));
                ui.Column(fact, 28, 8);
                ui.Text(fact, "WHAT THIS WAS", 26, true, TextAlignmentOptions.TopLeft, new Color(0.35f, 0.3f, 0.25f));
                ui.Text(fact, item.Fact, 38, true, TextAlignmentOptions.TopLeft, new Color(0.12f, 0.1f, 0.08f));
            }
            else if (item.IsHybrid)
            {
                var note = ui.Panel_("NoFact", panel, UIKit.PanelLight);
                ui.Column(note, 28, 8);
                ui.Text(note, "This object was made by the transmission and never existed above ground. There is nothing true to say about it.", 34, false, TextAlignmentOptions.TopLeft, new Color(0.75f, 0.75f, 0.75f));
            }

            // Fiction panel: sans, dark.
            if (!string.IsNullOrEmpty(item.Fiction))
            {
                var fic = ui.Panel_("Fiction", panel, UIKit.PanelLight);
                ui.Column(fic, 28, 8);
                ui.Text(fic, "THE ANVIL SAYS", 26, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
                ui.Text(fic, item.Fiction, 38, false, TextAlignmentOptions.TopLeft);
            }
        }
    }
}
