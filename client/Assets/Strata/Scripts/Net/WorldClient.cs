using System;
using System.Collections;
using System.Globalization;
using System.Text;
using Newtonsoft.Json;
using UnityEngine;
using UnityEngine.Networking;

namespace Strata.Net
{
    /// <summary>
    /// Thin HTTP client for worldd. Coroutine based so it runs on the main thread with no
    /// threading in a throwaway prototype. Every call reports either a typed result or an
    /// error string; the caller decides what to show.
    /// </summary>
    public sealed class WorldClient
    {
        public string BaseUrl;
        public string DeviceId;
        public float TimeoutSeconds = 8f;

        public WorldClient(string baseUrl, string deviceId)
        {
            BaseUrl = baseUrl;
            DeviceId = deviceId;
        }

        public sealed class Reply<T>
        {
            public T Value;
            public string Error;
            public long Status;
            public bool Ok => Error == null;
        }

        public IEnumerator Lookup(double lat, double lon, Action<Reply<LookupOut>> done)
        {
            var url = $"{BaseUrl}/v0/lookup?lat={F(lat)}&lon={F(lon)}";
            yield return Get(url, done);
        }

        public IEnumerator Spawns(double lat, double lon, Action<Reply<SpawnsOut>> done)
        {
            var url = $"{BaseUrl}/v0/spawns?lat={F(lat)}&lon={F(lon)}";
            yield return Get(url, done);
        }

        public IEnumerator Collapse(CollapseIn body, Action<Reply<CollapseResult>> done)
        {
            var url = $"{BaseUrl}/v0/collapse";
            var json = JsonConvert.SerializeObject(body);
            using var req = new UnityWebRequest(url, "POST");
            req.uploadHandler = new UploadHandlerRaw(Encoding.UTF8.GetBytes(json));
            req.downloadHandler = new DownloadHandlerBuffer();
            req.SetRequestHeader("Content-Type", "application/json");
            req.SetRequestHeader("X-Strata-Device", DeviceId);
            req.timeout = Mathf.CeilToInt(TimeoutSeconds);
            yield return req.SendWebRequest();
            done(Parse<CollapseResult>(req));
        }

        public IEnumerator Health(Action<Reply<object>> done)
        {
            yield return Get($"{BaseUrl}/healthz", done);
        }

        private IEnumerator Get<T>(string url, Action<Reply<T>> done)
        {
            using var req = UnityWebRequest.Get(url);
            req.SetRequestHeader("X-Strata-Device", DeviceId);
            req.timeout = Mathf.CeilToInt(TimeoutSeconds);
            yield return req.SendWebRequest();
            done(Parse<T>(req));
        }

        private static Reply<T> Parse<T>(UnityWebRequest req)
        {
            var r = new Reply<T> { Status = req.responseCode };
            if (req.result != UnityWebRequest.Result.Success && req.result != UnityWebRequest.Result.ProtocolError)
            {
                r.Error = req.error ?? "network error";
                return r;
            }
            var text = req.downloadHandler?.text ?? "";
            if (req.responseCode >= 400)
            {
                try
                {
                    var e = JsonConvert.DeserializeObject<ErrorOut>(text);
                    r.Error = e?.Error ?? $"HTTP {req.responseCode}";
                }
                catch
                {
                    r.Error = $"HTTP {req.responseCode}";
                }
                return r;
            }
            try
            {
                r.Value = JsonConvert.DeserializeObject<T>(text);
            }
            catch (Exception ex)
            {
                r.Error = "bad json: " + ex.Message;
            }
            return r;
        }

        private static string F(double v) => v.ToString("F6", CultureInfo.InvariantCulture);
    }
}
