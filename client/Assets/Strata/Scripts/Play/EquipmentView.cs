using System;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// What is worn, slot by slot, and the four stats it adds up to (PLAN §8). Tapping a slot
    /// offers the bag's items for that slot. Pure display over PlayerStore; nothing here is
    /// sent to the server.
    /// </summary>
    public sealed class EquipmentView
    {
        private readonly UIKit _ui;
        private readonly PlayerStore _store;
        private RectTransform _panel;
        private Action _closed;

        public EquipmentView(UIKit ui, PlayerStore store) { _ui = ui; _store = store; }

        public void Show(Action closed)
        {
            _closed = closed;
            Build();
        }

        private void Build()
        {
            if (_panel != null) UnityEngine.Object.Destroy(_panel.gameObject);
            _panel = _ui.Full("Equipment", _ui.Root, UIKit.PanelOpaque);
            _ui.Column(_panel, 40, 16);
            _ui.Header(_panel, "Gear", Close);

            var content = _ui.Scroll(_panel);

            // Stats block.
            var stats = _ui.Framed("Stats", content, UIKit.PanelLight);
            _ui.Column(stats, 28, 6);
            _ui.Title(stats, "FROM WHAT YOU WEAR", 24);
            var totals = _store.StatTotals();
            var row = _ui.Row(stats, 84, 12);
            foreach (var s in PlayerStore.Stats)
            {
                var cell = _ui.Framed("Stat " + s, row, UIKit.Panel);
                _ui.Column(cell, 12, 0);
                _ui.Title(cell, CivPalette.StatLabel(s).ToUpper(), 22, TextAlignmentOptions.Center, UIKit.Muted);
                _ui.Title(cell, totals.TryGetValue(s, out var v) ? v.ToString("0") : "0", 44, TextAlignmentOptions.Center, UIKit.Ink);
            }
            var extra = new List<string>();
            foreach (var kv in totals) if (Array.IndexOf(PlayerStore.Stats, kv.Key) < 0 && kv.Value != 0) extra.Add($"{kv.Key} +{kv.Value:0}");
            if (extra.Count > 0) _ui.Text(stats, string.Join("   ", extra), 30, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));

            // Slots.
            foreach (var slot in PlayerStore.Slots)
            {
                var item = _store.EquippedIn(slot);
                var label = item == null
                    ? $"<color=#9A9A9A>{SlotName(slot)}</color>\n<size=70%><color=#777777>empty</color></size>"
                    : $"<color=#9A9A9A>{SlotName(slot)}</color>\n<size=85%>{CivPalette.Coloured(item.Civ)}  {item.Name}  <color=#AAAAAA>· {item.TierName}{Affixes(item)}</color></size>";
                var s = slot;
                _ui.Button_(content, label, UIKit.PanelLight, () => Choose(s), 130, TextAlignmentOptions.Left, UIKit.Ink);
            }
        }

        private static string Affixes(Item item)
        {
            if (item.Affixes == null || item.Affixes.Count == 0) return "";
            var parts = new List<string>();
            foreach (var a in item.Affixes) parts.Add($"{CivPalette.StatLabel(a.K)} +{a.V:0}");
            return " · " + string.Join(" ", parts);
        }

        public static string SlotName(string slot) => slot switch
        {
            "main" => "MAIN HAND", "off" => "OFF HAND", "body" => "BODY", "head" => "HEAD",
            "charm1" => "CHARM I", "charm2" => "CHARM II", "relic" => "RELIC", _ => slot.ToUpper(),
        };

        private void Choose(string slot)
        {
            var chooser = _ui.Full("Choose " + slot, _ui.Root, UIKit.PanelOpaque);
            _ui.Column(chooser, 40, 16);
            _ui.Header(chooser, SlotName(slot), () => { UnityEngine.Object.Destroy(chooser.gameObject); Build(); });
            var content = _ui.Scroll(chooser);
            var options = _store.ItemsForSlot(slot);
            if (options.Count == 0) _ui.Text(content, "nothing in the bag fits this slot yet", 36, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            foreach (var it in options)
            {
                var item = it;
                var worn = _store.IsEquipped(item) ? "  <color=#E9C46A>worn</color>" : "";
                _ui.Button_(content, $"{CivPalette.Coloured(item.Civ)}  {item.Name}{worn}\n<size=75%><color=#AAAAAA>{item.TierName} · {item.Authenticity}{Affixes(item)}</color></size>", UIKit.PanelLight, () =>
                {
                    if (slot.StartsWith("charm")) { _store.Unequip(item); _store.equipped[slot] = item.Id; _store.Save(); }
                    else _store.Equip(item);
                    UnityEngine.Object.Destroy(chooser.gameObject);
                    Build();
                }, 120, TextAlignmentOptions.Left, UIKit.Ink);
            }
            if (_store.EquippedIn(slot) != null)
                _ui.Button_(chooser, "Take it off", UIKit.PanelLight, () => { _store.UnequipSlot(slot); UnityEngine.Object.Destroy(chooser.gameObject); Build(); }, 90);
        }

        private void Close()
        {
            if (_panel != null) UnityEngine.Object.Destroy(_panel.gameObject);
            _panel = null;
            _closed?.Invoke();
        }
    }
}
