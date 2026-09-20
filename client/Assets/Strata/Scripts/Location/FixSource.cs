using System;
using System.Collections;
using UnityEngine;
#if UNITY_ANDROID
using UnityEngine.Android;
#endif

namespace Strata.Location
{
    public struct Fix
    {
        public double Lat, Lon;
        public float AccuracyM;
        public double Timestamp; // unix seconds
    }

    /// <summary>
    /// Location fixes from Unity's LocationService on the device, or from a caller-supplied
    /// fallback (the GO Map editor simulation) when the service is unavailable. Polls on a
    /// fixed interval and raises OnFix. Asks for the fine location permission on Android.
    /// </summary>
    public sealed class FixSource : MonoBehaviour
    {
        public float intervalSeconds = 2f;
        public Func<(bool ok, double lat, double lon)> fallback;

        public event Action<Fix> OnFix;
        public event Action<string> OnStatus;

        public Fix? Last { get; private set; }
        public bool Running { get; private set; }

        public void Begin()
        {
            if (!Running) StartCoroutine(Run());
        }

        private IEnumerator Run()
        {
            Running = true;
#if UNITY_ANDROID && !UNITY_EDITOR
            if (!Permission.HasUserAuthorizedPermission(Permission.FineLocation))
            {
                Status("asking for location permission");
                Permission.RequestUserPermission(Permission.FineLocation);
                float waited = 0;
                while (!Permission.HasUserAuthorizedPermission(Permission.FineLocation) && waited < 30f)
                {
                    waited += Time.deltaTime;
                    yield return null;
                }
            }
#endif
            bool device = false;
            if (Input.location.isEnabledByUser)
            {
                Input.location.Start(5f, 5f);
                float waited = 0;
                while (Input.location.status == LocationServiceStatus.Initializing && waited < 20f)
                {
                    waited += 0.5f;
                    yield return new WaitForSeconds(0.5f);
                }
                device = Input.location.status == LocationServiceStatus.Running;
                Status(device ? "location: device GPS" : "location: device service failed, using map simulation");
            }
            else
            {
                Status("location: disabled on device, using map simulation");
            }

            while (true)
            {
                if (device && Input.location.status == LocationServiceStatus.Running)
                {
                    var d = Input.location.lastData;
                    Emit(d.latitude, d.longitude, d.horizontalAccuracy, d.timestamp);
                }
                else if (fallback != null)
                {
                    var (ok, lat, lon) = fallback();
                    if (ok) Emit(lat, lon, 10f, DateTimeOffset.UtcNow.ToUnixTimeSeconds());
                }
                yield return new WaitForSeconds(intervalSeconds);
            }
        }

        private void Emit(double lat, double lon, float acc, double ts)
        {
            if (lat == 0 && lon == 0) return;
            var f = new Fix { Lat = lat, Lon = lon, AccuracyM = acc, Timestamp = ts };
            Last = f;
            OnFix?.Invoke(f);
        }

        private void Status(string s) => OnStatus?.Invoke(s);

        private void OnDestroy()
        {
            if (Input.location.status == LocationServiceStatus.Running) Input.location.Stop();
        }
    }
}
