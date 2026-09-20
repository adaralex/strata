using System;
using TMPro;
using UnityEngine;
using Strata.Net;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The bag: every item the server has ever handed this phone, newest first. Tapping one
    /// opens its card (the same fact/fiction layout as the drop) with equip controls.
    /// </summary>
    public sealed class InventoryView
    {
        private readonly UIKit _ui;
        private readonly PlayerStore _store;
        private RectTransform _panel;
        private Action _closed;

        public InventoryView(UIKit ui, PlayerStore store) { _ui = ui; _store = store; }

        public void Show(Action closed)
        {
            _closed = closed;
            Build();
        }

        private void Build()
        {
            if (_panel != null) UnityEngine.Object.Destroy(_panel.gameObject);
            _panel = _ui.Full("Inventory", _ui.Root, UIKit.PanelOpaque);
            _ui.Column(_panel, 40, 16);
            var items = _store.ItemsByRecency();
            _ui.Header(_panel, $"Bag  <size=60%><color=#AAAAAA>{items.Count} thing{(items.Count == 1 ? "" : "s")}</color></size>", Close);
            var content = _ui.Scroll(_panel);
            if (items.Count == 0)
                _ui.Text(content, "Nothing yet. Collapse a monster within 40 m and what the server sends back lands here.", 36, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            foreach (var it in items)
            {
                var item = it;
                var worn = _store.IsEquipped(item) ? "  <color=#E9C46A>worn</color>" : "";
                var hybrid = item.IsHybrid ? "  <color=#AAAAAA>hybrid</color>" : "";
                _ui.Button_(content, $"{CivPalette.Coloured(item.Civ)}  {item.Name}{worn}{hybrid}\n<size=75%><color=#AAAAAA>{item.TierName} · {item.Authenticity}{(string.IsNullOrEmpty(item.Slot) ? "" : " · " + item.Slot)}</color></size>", UIKit.PanelLight, () => Detail(item), 120, TextAlignmentOptions.Left, UIKit.Ink);
            }
        }

        private void Detail(Item item)
        {
            var card = _ui.Full("Item", _ui.Root, UIKit.PanelOpaque);
            _ui.Column(card, 40, 16);
            _ui.Header(card, "", () => { UnityEngine.Object.Destroy(card.gameObject); Build(); });
            var content = _ui.Scroll(card);
            ItemCard.Describe(_ui, content, item, null);
            if (_store.CanEquip(item))
            {
                var worn = _store.IsEquipped(item);
                _ui.Button_(card, worn ? "Take it off" : "Wear it", UIKit.Accent, () =>
                {
                    if (worn) _store.Unequip(item); else _store.Equip(item);
                    UnityEngine.Object.Destroy(card.gameObject);
                    Build();
                });
            }
            else if (item.Kind == "edible")
                _ui.Text(card, "edibles are not worn; eating is not in phase 0", 30, false, TextAlignmentOptions.Center, new Color(0.6f, 0.6f, 0.6f));
        }

        private void Close()
        {
            if (_panel != null) UnityEngine.Object.Destroy(_panel.gameObject);
            _panel = null;
            _closed?.Invoke();
        }
    }
}
