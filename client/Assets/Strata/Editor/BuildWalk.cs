using System.IO;
using UnityEditor;
using UnityEditor.Build;
using UnityEditor.Build.Reporting;
using UnityEngine;

namespace Strata.EditorTools
{
    /// <summary>
    /// Builds the walk-test APK: Strata > Build Walk APK. Output goes to Builds/strata-walk.apk
    /// next to Assets, and Builds/last-build.txt records the summary for scripts that wait on it.
    /// </summary>
    public static class BuildWalk
    {
        public const string Scene = "Assets/Strata/Scenes/Walk.unity";

        public static string OutputDir => Path.Combine(Path.GetDirectoryName(Application.dataPath), "Builds");
        public static string ApkPath => Path.Combine(OutputDir, "strata-walk.apk");

        [MenuItem("Strata/Build Walk APK")]
        public static void Build()
        {
            PlayerSettings.SetApplicationIdentifier(NamedBuildTarget.Android, "com.strata.walk");
            PlayerSettings.productName = "Strata";
            EditorUserBuildSettings.buildAppBundle = false;
            Directory.CreateDirectory(OutputDir);
            var summaryPath = Path.Combine(OutputDir, "last-build.txt");
            if (File.Exists(summaryPath)) File.Delete(summaryPath);
            if (File.Exists(ApkPath)) File.Delete(ApkPath);

            Debug.Log("[Strata build] starting " + ApkPath);
            var report = BuildPipeline.BuildPlayer(new BuildPlayerOptions
            {
                scenes = new[] { Scene },
                locationPathName = ApkPath,
                target = BuildTarget.Android,
                options = BuildOptions.None,
            });
            var s = report.summary;
            var line = $"{s.result} errors={s.totalErrors} warnings={s.totalWarnings} size={s.totalSize} time={s.totalTime}";
            Debug.Log("[Strata build] " + line);
            File.WriteAllText(summaryPath, line);
        }

        /// <summary>Runs Build once the editor is idle (not playing, not compiling).</summary>
        public static void BuildWhenIdle()
        {
            EditorApplication.CallbackFunction tick = null;
            tick = () =>
            {
                if (EditorApplication.isPlaying || EditorApplication.isPlayingOrWillChangePlaymode || EditorApplication.isCompiling) return;
                EditorApplication.update -= tick;
                Build();
            };
            EditorApplication.update += tick;
        }
    }
}
