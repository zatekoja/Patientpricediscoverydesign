import { useState } from "react";
import { Navigation } from "lucide-react";
import { APIProvider, Map, AdvancedMarker, Pin, InfoWindow } from "@vis.gl/react-google-maps";
import { API_BASE_URL } from "../../lib/api";
import type { UIFacility } from "../../lib/mappers";

const GOOGLE_MAPS_API_KEY = import.meta.env.VITE_GOOGLE_MAPS_API_KEY || "";

interface MapViewProps {
  facilities: UIFacility[];
  center: { lat: number; lon: number };
  onSelectFacility: (facility: UIFacility) => void;
}

export function MapView({ facilities, center, onSelectFacility }: MapViewProps) {
  const [selectedMarker, setSelectedMarker] = useState<UIFacility | null>(null);
  const [hoveredFacilityId, setHoveredFacilityId] = useState<string | null>(null);

  // Fallback to static map if no API key is provided
  if (!GOOGLE_MAPS_API_KEY) {
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
              <span className="text-xs text-gray-600">Static map (Add VITE_GOOGLE_MAPS_API_KEY for interactive map).</span>
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
                        <span className="font-semibold text-gray-900">Pricing in details</span>
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

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="flex">
        {/* Interactive Map Area */}
        <div className="flex-1 relative bg-gray-100" style={{ height: "calc(100vh - 250px)" }}>
          <APIProvider apiKey={GOOGLE_MAPS_API_KEY}>
            <Map
              defaultCenter={{ lat: center.lat, lng: center.lon }}
              center={{ lat: center.lat, lng: center.lon }}
              defaultZoom={12}
              zoom={12}
              gestureHandling={"greedy"}
              disableDefaultUI={false}
              mapId={"DEMO_MAP_ID"} // Required for AdvancedMarker, can be any string for basic usage or real ID from Google Console
              className="w-full h-full"
            >
              <AdvancedMarker position={{ lat: center.lat, lng: center.lon }}>
                <Pin background={"#2563EB"} glyphColor={"white"} borderColor={"#1E40AF"}>
                   <Navigation className="w-4 h-4 text-white" />
                </Pin>
              </AdvancedMarker>

              {facilities.map((facility) => {
                 if (facility.lat == null || facility.lon == null) return null;
                 const isHovered = hoveredFacilityId === facility.id;
                 const isSelected = selectedMarker?.id === facility.id;

                 return (
                  <AdvancedMarker
                    key={facility.id}
                    position={{ lat: facility.lat, lng: facility.lon }}
                    onClick={() => {
                        setSelectedMarker(facility);
                        onSelectFacility(facility);
                    }}
                    zIndex={isSelected || isHovered ? 10 : 1}
                  >
                     <Pin
                       background={isSelected ? "#DC2626" : isHovered ? "#EF4444" : "#EA4335"}
                       glyphColor={"white"}
                       borderColor={isSelected ? "#991B1B" : "#B91C1C"}
                       scale={isSelected || isHovered ? 1.2 : 1}
                     />
                  </AdvancedMarker>
                 );
              })}

              {selectedMarker && selectedMarker.lat && selectedMarker.lon && (
                <InfoWindow
                  position={{ lat: selectedMarker.lat, lng: selectedMarker.lon }}
                  onCloseClick={() => setSelectedMarker(null)}
                  pixelOffset={[0, -30]}
                >
                  <div className="p-1 min-w-[200px]">
                    <h3 className="font-semibold text-gray-900 text-sm mb-1">{selectedMarker.name}</h3>
                    <p className="text-xs text-gray-600 mb-1">{selectedMarker.address}</p>
                    <div className="flex items-center justify-between text-xs">
                      <span className="font-medium text-blue-600">Pricing in details</span>
                      <span>{selectedMarker.distanceKm.toFixed(1)} km</span>
                    </div>
                     <button
                        className="mt-2 w-full text-xs bg-blue-600 text-white py-1.5 px-3 rounded hover:bg-blue-700 transition-colors"
                        onClick={() => onSelectFacility(selectedMarker)}
                      >
                        View Details
                      </button>
                  </div>
                </InfoWindow>
              )}
            </Map>
          </APIProvider>
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
                  className={`rounded-lg p-4 cursor-pointer transition-colors ${
                      selectedMarker?.id === facility.id ? "bg-blue-50 border border-blue-200" : "bg-gray-50 hover:bg-gray-100"
                  }`}
                  onClick={() => {
                      onSelectFacility(facility);
                      setSelectedMarker(facility);
                  }}
                  onMouseEnter={() => setHoveredFacilityId(facility.id)}
                  onMouseLeave={() => setHoveredFacilityId(null)}
                >
                  <div className="flex items-start justify-between mb-2">
                    <h4 className="font-semibold text-gray-900 text-sm">
                      {facility.name}
                    </h4>
                     <div className="flex items-center gap-2">
                        {facility.capacityStatus && (
                          <div
                            className={`w-2 h-2 rounded-full ${
                              facility.capacityStatus === "Available"
                                ? "bg-green-600"
                                : "bg-yellow-600"
                            }`}
                          />
                        )}
                     </div>
                  </div>
                  <div className="space-y-1 text-xs text-gray-600">
                    <div className="flex items-center justify-between">
                      <span>{facility.distanceKm.toFixed(2)} km away</span>
                      <span className="font-semibold text-gray-900">Pricing in details</span>
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
