using System;
using System.IO;
using Newtonsoft.Json;
using TMPro;
using UnityEngine;
using Strata.UI;

namespace Strata.Play
{
    /// <summary>
    /// The end of the walk: the log summary and three questions. Saved as JSON to the
    /// device's persistent data path and offered through the share sheet. That file is
    /// phase 0's gate: did it feel like anything?
    /// </summary>
    public sealed class Debrief
    {
        private readonly UIKit _ui;

        public Debrief(UIKit ui) { _ui = ui; }

        [Serializable]
        public sealed class Answers
        {
            public string differentAnywhere;
            public string remembered;
            public string walkAgain;
            public string freeText;
        }

        public void Show(WalkLog log, Action closed)
        {
            var panel = _ui.Full("Debrief", _ui.Root, UIKit.Panel);
            var col = _ui.Column(panel, 40, 18);
            col.childAlignment = TextAnchor.UpperLeft;
            _ui.Title(panel, "End of walk", 52);
            _ui.Text(panel, log.Summary(), 40);

            _ui.Text(panel, "Did the ground feel different anywhere? Where?", 36, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            var q1 = _ui.Input(panel, "", "…");
            _ui.Text(panel, "Which fight or drop do you remember?", 36, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            var q2 = _ui.Input(panel, "", "…");
            _ui.Text(panel, "Would you walk it again? Why?", 36, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            var q3 = _ui.Input(panel, "", "…");
            _ui.Text(panel, "Anything else", 36, false, TextAlignmentOptions.TopLeft, UIKit.Accent);
            var q4 = _ui.Input(panel, "", "…");

            var status = _ui.Text(panel, "", 32, false, TextAlignmentOptions.TopLeft, new Color(0.7f, 0.7f, 0.7f));
            _ui.Button_(panel, "Save and share", UIKit.Accent, () =>
            {
                log.endedAt = DateTimeOffset.UtcNow.ToString("o");
                var answers = new Answers { differentAnywhere = q1.text, remembered = q2.text, walkAgain = q3.text, freeText = q4.text };
                var path = Save(log, answers);
                status.text = "saved to " + path;
                Share(path);
            });
            _ui.Button_(panel, "Back to the map", UIKit.PanelLight, () =>
            {
                UnityEngine.Object.Destroy(panel.gameObject);
                closed?.Invoke();
            }, 90);
        }

        private static string Save(WalkLog log, Answers answers)
        {
            var dir = Application.persistentDataPath;
            var path = Path.Combine(dir, $"strata-walk-{DateTime.UtcNow:yyyyMMdd-HHmmss}.json");
            var doc = new { log, answers, app = Application.version, platform = Application.platform.ToString() };
            File.WriteAllText(path, JsonConvert.SerializeObject(doc, Formatting.Indented));
            return path;
        }

        private static void Share(string path)
        {
#if UNITY_ANDROID && !UNITY_EDITOR
            try
            {
                using var intentClass = new AndroidJavaClass("android.content.Intent");
                using var intent = new AndroidJavaObject("android.content.Intent");
                intent.Call<AndroidJavaObject>("setAction", intentClass.GetStatic<string>("ACTION_SEND"));
                intent.Call<AndroidJavaObject>("setType", "text/plain");
                intent.Call<AndroidJavaObject>("putExtra", intentClass.GetStatic<string>("EXTRA_TEXT"), File.ReadAllText(path));
                using var unity = new AndroidJavaClass("com.unity3d.player.UnityPlayer");
                using var activity = unity.GetStatic<AndroidJavaObject>("currentActivity");
                using var chooser = intentClass.CallStatic<AndroidJavaObject>("createChooser", intent, "Share the walk");
                activity.Call("startActivity", chooser);
            }
            catch (Exception e)
            {
                Debug.LogWarning("share failed: " + e.Message);
            }
#else
            Debug.Log("walk saved: " + path);
#endif
        }
    }
}
