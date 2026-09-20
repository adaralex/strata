using System.Collections.Generic;
using UnityEngine;

namespace Strata.UI
{
    /// <summary>Placeholder colours per civilization. No art in phase 0; these only need to be distinct.</summary>
    public static class CivPalette
    {
        private static readonly Dictionary<string, Color> Colours = new Dictionary<string, Color>
        {
            { "etruria", Hex("C8553D") }, { "minoa", Hex("2A9D8F") }, { "kemet", Hex("E9C46A") },
            { "sumer", Hex("A98467") }, { "phoenicia", Hex("6A4C93") }, { "scythia", Hex("D4A017") },
            { "hallstatt", Hex("3A7D44") }, { "meluhha", Hex("B5651D") }, { "shang", Hex("1D3557") },
            { "nok", Hex("8D5524") }, { "olmec", Hex("2F6B3A") }, { "chavin", Hex("B23A48") },
            { "lapita", Hex("0077B6") }, { "hopewell", Hex("C77DFF") }, { "aksum", Hex("E76F51") },
        };

        /// <summary>The colour of the residual: the eleven civilizations folded into one number.</summary>
        public static readonly Color Others = new Color(0.45f, 0.45f, 0.45f);

        public static Color Of(string civ) => civ != null && Colours.TryGetValue(civ, out var c) ? c : Color.gray;

        public static string Label(string civ) => string.IsNullOrEmpty(civ) ? "?" : char.ToUpper(civ[0]) + civ.Substring(1);

        /// <summary>The label wrapped in a TextMeshPro colour tag in the civilization's colour.</summary>
        public static string Coloured(string civ) => $"<color=#{ColorUtility.ToHtmlStringRGB(Of(civ))}>{Label(civ)}</color>";

        /// <summary>Display names for the plan's four stats (PLAN §8); other keys pass through.</summary>
        public static string StatLabel(string key) => key switch
        {
            "might" => "Might", "ward" => "Ward", "resonance" => "Resonance", "fortune" => "Fortune",
            null => "?", _ => key,
        };

        private static Color Hex(string h)
        {
            ColorUtility.TryParseHtmlString("#" + h, out var c);
            return c;
        }
    }
}
