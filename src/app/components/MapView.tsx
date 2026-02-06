import { MapPin, Navigation } from "lucide-react";

interface MapViewProps {
  onSelectFacility: (facility: any) => void;
}

// Mock facilities with coordinates for map view
const facilities = [
  {
    id: 1,
    name: "St. Mary's Medical Center",
    distance: 2.3,
    price: 450,
    nextAvailable: "Today, 3:00 PM",
    capacity: "Available",
    lat: 40.7580,
    lng: -73.9855,
  },
  {
    id: 2,
    name: "Central Imaging & Diagnostics",
    distance: 3.7,
    price: 350,
    nextAvailable: "Tomorrow, 9:00 AM",
    capacity: "Limited",
    lat: 40.7489,
    lng: -73.9680,
  },
  {
    id: 3,
    name: "Regional Emergency Hospital",
    distance: 5.1,
    price: 520,
    nextAvailable: "Today, 5:30 PM",
    capacity: "Available",
    lat: 40.7789,
    lng: -73.9750,
  },
  {
    id: 4,
    name: "Advanced Medical Imaging",
    distance: 7.8,
    price: 295,
    nextAvailable: "Feb 8, 10:00 AM",
    capacity: "Available",
    lat: 40.7350,
    lng: -73.9950,
  },
  {
    id: 5,
    name: "Community Health Center",
    distance: 4.2,
    price: 380,
    nextAvailable: "Feb 7, 2:00 PM",
    capacity: "Limited",
    lat: 40.7650,
    lng: -73.9550,
  },
  {
    id: 6,
    name: "University Medical Center",
    distance: 9.5,
    price: 550,
    nextAvailable: "Tomorrow, 11:00 AM",
    capacity: "Available",
    lat: 40.7280,
    lng: -73.9800,
  },
];

export function MapView({ onSelectFacility }: MapViewProps) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="flex">
        {/* Map Area */}
        <div className="flex-1 relative bg-gray-100" style={{ height: "calc(100vh - 250px)" }}>
          {/* Placeholder map background */}
          <div className="absolute inset-0 bg-gradient-to-br from-blue-50 to-gray-100">
            {/* Grid lines to simulate map */}
            <svg className="w-full h-full opacity-20">
              <defs>
                <pattern
                  id="grid"
                  width="40"
                  height="40"
                  patternUnits="userSpaceOnUse"
                >
                  <path
                    d="M 40 0 L 0 0 0 40"
                    fill="none"
                    stroke="gray"
                    strokeWidth="0.5"
                  />
                </pattern>
              </defs>
              <rect width="100%" height="100%" fill="url(#grid)" />
            </svg>

            {/* User location marker */}
            <div
              className="absolute bg-blue-600 rounded-full p-3 shadow-lg"
              style={{ top: "50%", left: "50%", transform: "translate(-50%, -50%)" }}
            >
              <Navigation className="w-5 h-5 text-white" />
            </div>

            {/* Facility markers */}
            {facilities.map((facility, index) => {
              // Position markers around the center
              const angle = (index * 2 * Math.PI) / facilities.length;
              const radius = 150;
              const x = 50 + Math.cos(angle) * radius;
              const y = 50 + Math.sin(angle) * radius;

              return (
                <div
                  key={facility.id}
                  className="absolute cursor-pointer group"
                  style={{
                    left: `${x}%`,
                    top: `${y}%`,
                    transform: "translate(-50%, -50%)",
                  }}
                  onClick={() => onSelectFacility(facility)}
                >
                  <div
                    className={`rounded-full p-2 shadow-lg transition-transform group-hover:scale-110 ${
                      facility.capacity === "Available"
                        ? "bg-green-600"
                        : "bg-yellow-600"
                    }`}
                  >
                    <MapPin className="w-5 h-5 text-white" />
                  </div>
                  {/* Tooltip on hover */}
                  <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none">
                    <div className="bg-gray-900 text-white text-xs rounded-lg px-3 py-2 whitespace-nowrap">
                      <div className="font-semibold">{facility.name}</div>
                      <div className="text-gray-300 mt-1">
                        {facility.distance} mi • ${facility.price}
                      </div>
                    </div>
                    <div className="w-2 h-2 bg-gray-900 absolute top-full left-1/2 -translate-x-1/2 -translate-y-1/2 rotate-45" />
                  </div>
                </div>
              );
            })}
          </div>

          {/* Map Legend */}
          <div className="absolute bottom-4 left-4 bg-white rounded-lg shadow-lg p-4">
            <h4 className="font-semibold text-gray-900 mb-3">Legend</h4>
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 bg-blue-600 rounded-full" />
                <span className="text-sm text-gray-700">Your Location</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 bg-green-600 rounded-full" />
                <span className="text-sm text-gray-700">Available</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 bg-yellow-600 rounded-full" />
                <span className="text-sm text-gray-700">Limited Capacity</span>
              </div>
            </div>
          </div>

          {/* Zoom Controls */}
          <div className="absolute top-4 right-4 bg-white rounded-lg shadow-lg overflow-hidden">
            <button className="block px-4 py-2 hover:bg-gray-100 border-b border-gray-200">
              +
            </button>
            <button className="block px-4 py-2 hover:bg-gray-100">−</button>
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
                    <div
                      className={`w-2 h-2 rounded-full mt-1 ${
                        facility.capacity === "Available"
                          ? "bg-green-600"
                          : "bg-yellow-600"
                      }`}
                    />
                  </div>
                  <div className="space-y-1 text-xs text-gray-600">
                    <div className="flex items-center justify-between">
                      <span>{facility.distance} miles away</span>
                      <span className="font-semibold text-gray-900">
                        ${facility.price}
                      </span>
                    </div>
                    <div className="text-green-700">
                      Next: {facility.nextAvailable}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
