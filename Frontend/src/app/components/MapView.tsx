import { Navigation } from "lucide-react";
import { API_BASE_URL } from "../../lib/api";
import type { UIFacility } from "../../lib/mappers";

interface MapViewProps {
  facilities: UIFacility[];
  center: { lat: number; lon: number };
  onSelectFacility: (facility: UIFacility) => void;
}

export function MapView({ facilities, center, onSelectFacility }: MapViewProps) {
  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : currency ? `${currency} ` : "₦";
    return `${symbol}${Math.round(value).toLocaleString()}`;
  };
  const mapParams = new URLSearchParams({
    center: `${center.lat},${center.lon}`,
    zoom: "12",
    size: "640x360",
    scale: "1",
  });

  facilities.forEach((facility, index) => {
    if (facility.lat == null || facility.lon == null) {
      return;
    }
    const label = String.fromCharCode(65 + (index % 26));
    mapParams.append("markers", `color:red|label:${label}|${facility.lat},${facility.lon}`);
  });

  const mapSrc = `${API_BASE_URL}/maps/static?${mapParams.toString()}`;

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="flex">
        {/* Map Area */}
        <div className="flex-1 relative bg-gray-100" style={{ height: "calc(100vh - 250px)" }}>
          <img
            src={mapSrc}
            alt="Facility map"
            className="absolute inset-0 h-full w-full object-cover"
            loading="lazy"
          />

          {/* User location marker */}
          <div
            className="absolute bg-blue-600 rounded-full p-3 shadow-lg"
            style={{ top: "50%", left: "50%", transform: "translate(-50%, -50%)" }}
          >
            <Navigation className="w-5 h-5 text-white" />
          </div>

          <div className="absolute bottom-4 left-4 bg-white rounded-lg shadow-lg px-3 py-2">
            <span className="text-xs text-gray-600">Static map to save data.</span>
          </div>
        </div>

        {/* Sidebar with facility list */}
        <div className="w-96 border-l border-gray-200 overflow-y-auto" style={{ height: "calc(100vh - 250px)" }}>
          <div className="p-4">
            <h3 className="font-semibold text-gray-900 mb-4">
              {facilities.length} Facilities Nearby
            </h3>
            <div className="space-y-3">
              {facilities.map((facility) => (
                <div
                  key={facility.id}
                  className="bg-gray-50 rounded-lg p-4 cursor-pointer hover:bg-gray-100 transition-colors"
                  onClick={() => onSelectFacility(facility)}
                >
                  <div className="flex items-start justify-between mb-2">
                    <h4 className="font-semibold text-gray-900 text-sm">
                      {facility.name}
                    </h4>
                    {facility.capacityStatus && (
                      <div
                        className={`w-2 h-2 rounded-full mt-1 ${
                          facility.capacityStatus === "Available"
                            ? "bg-green-600"
                            : "bg-yellow-600"
                        }`}
                      />
                    )}
                  </div>
                  <div className="space-y-1 text-xs text-gray-600">
                    <div className="flex items-center justify-between">
                      <span>{facility.distanceKm.toFixed(2)} km away</span>
                      <span className="font-semibold text-gray-900">
                        {facility.priceMin != null
                          ? formatCurrency(facility.priceMin, facility.currency)
                          : "Price N/A"}
                      </span>
                    </div>
                    {facility.nextAvailableAt && (
                      <div className="text-green-700">
                        Next: {new Date(facility.nextAvailableAt).toLocaleDateString("en-NG", { month: "short", day: "numeric" })}
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {facilities.length === 0 && (
                <div className="text-sm text-gray-500">
                  No facilities found for this area.
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
