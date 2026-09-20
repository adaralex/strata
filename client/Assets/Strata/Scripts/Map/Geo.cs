using System;

namespace Strata.Map
{
    /// <summary>Client-side geodesy for display only. The server decides ranges.</summary>
    public static class Geo
    {
        private const double R = 6371008.8;

        public static double DistanceM(double lat1, double lon1, double lat2, double lon2)
        {
            double rad = Math.PI / 180;
            double dlat = (lat2 - lat1) * rad, dlon = (lon2 - lon1) * rad;
            double a = Math.Sin(dlat / 2) * Math.Sin(dlat / 2) +
                       Math.Cos(lat1 * rad) * Math.Cos(lat2 * rad) * Math.Sin(dlon / 2) * Math.Sin(dlon / 2);
            return 2 * R * Math.Asin(Math.Min(1, Math.Sqrt(a)));
        }
    }
}
