using System;
using System.Collections.Generic;
using TMPro;
using UnityEngine;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The record: fights, wins, drops, wins by civilization, then every monster this phone
    /// has fought with its counts and its Anvil line. The bestiary's real notes are server
    /// content and are not in the spawn payload, so nothing here claims to be fact.
    /// </summary>
    public sealed class CodexView
    {
        private readonly UIKit _ui;
        private readonly PlayerStore _store;
        private RectTransform _panel;

        public CodexView(UIKit ui, PlayerStore store) { _ui = ui; _store = store; }

        public void Show(Action closed)
        {
            _panel = _ui.Full("Codex", _ui.Root, UIKit.PanelOpaque);
            _ui.Column(_panel, 40, 16);
            _ui.Header(_panel, "Codex", () => { UnityEngine.Object.Destroy(_panel.gameObject); _panel = null; closed?.Invoke(); });
            var content = _ui.Scroll(_panel);

            var b = _store.battle;
            var stats = _ui.Panel_("Battle", content, UIKit.PanelLight);
            _ui.Column(stats, 28, 6);
            _ui.Text(stats, "BATTLE RECORD", 26, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            var row = _ui.Row(stats, 70, 12);
            Cell(row, "fights", b.fights);
            Cell(row, "won", b.wins);
            Cell(row, "lost", b.losses);
            Cell(row, "auto", b.autoResolves);
            Cell(row, "drops", b.drops);
            if (b.winsByCiv.Count > 0)
            {
                var parts = new List<string>();
                var sorted = new List<KeyValuePair<string, int>>(b.winsByCiv);
                sorted.Sort((x, y) => y.Value.CompareTo(x.Value));
                foreach (var kv in sorted) parts.Add($"{CivPalette.Coloured(kv.Key)} {kv.Value}");
                _ui.Text(stats, "wins by civilization   " + string.Join("   ", parts), 30, false, TextAlignmentOptions.TopLeft, new Color(0.8f, 0.8f, 0.8f));
            }

            var entries = _store.CodexByRecency();
            _ui.Text(content, $"MONSTERS FOUGHT  <color=#AAAAAA>{entries.Count}</color>", 26, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            if (entries.Count == 0)
                _ui.Text(content, "No fights yet. Tap a marker within 40 m.", 36, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            foreach (var e in entries)
            {
                var card = _ui.Panel_("Monster", content, UIKit.PanelLight);
                _ui.Column(card, 24, 4);
                var rank = e.rank == "common" || string.IsNullOrEmpty(e.rank) ? "" : $"  <color=#E9C46A>{e.rank.ToUpper()}</color>";
                var hybrid = string.IsNullOrEmpty(e.secondary) ? "" : $"  <color=#AAAAAA>with {CivPalette.Label(e.secondary)}</color>";
                _ui.Text(card, $"{e.name}{rank}", 40);
                _ui.Text(card, $"{CivPalette.Coloured(e.civ)} · {e.tierName}{hybrid}", 30, false, TextAlignmentOptions.TopLeft, new Color(0.8f, 0.8f, 0.8f));
                _ui.Text(card, $"fought {e.fought} · won {e.won} · lost {e.lost}{(e.autoResolved > 0 ? $" · auto {e.autoResolved}" : "")}", 30, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
                if (!string.IsNullOrEmpty(e.fiction))
                    _ui.Text(card, e.fiction, 32, false, TextAlignmentOptions.TopLeft, new Color(0.85f, 0.85f, 0.85f));
            }
        }

        private void Cell(RectTransform row, string label, int value)
        {
            var cell = _ui.Panel_("Cell " + label, row, UIKit.Panel);
            _ui.Column(cell, 12, 0);
            _ui.Text(cell, label, 26, false, TextAlignmentOptions.Center, new Color(0.7f, 0.7f, 0.7f));
            _ui.Text(cell, value.ToString(), 44, false, TextAlignmentOptions.Center);
        }
    }
}
