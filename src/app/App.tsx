import { useState } from "react";
import { Search, MapPin, DollarSign, Calendar, Filter, X } from "lucide-react";
import { SearchResults } from "./components/SearchResults";
import { MapView } from "./components/MapView";
import { FacilityModal } from "./components/FacilityModal";

export default function App() {
  const [searchQuery, setSearchQuery] = useState("");
  const [location, setLocation] = useState("");
  const [showFilters, setShowFilters] = useState(false);
  const [viewMode, setViewMode] = useState<"list" | "map">("list");
  const [selectedFacility, setSelectedFacility] = useState<any>(null);

  // Filter states
  const [maxDistance, setMaxDistance] = useState("50");
  const [maxPrice, setMaxPrice] = useState("5000");
  const [insurance, setInsurance] = useState("");
  const [availability, setAvailability] = useState("any");

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-2xl font-bold text-blue-600">Open Health Initiative</h1>
            <p className="text-sm text-gray-600">Powered by Ateru</p>
          </div>

          {/* Search Bar */}
          <div className="flex gap-3 mb-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Search for procedures (e.g., MRI, CT Scan, X-Ray...)"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <div className="w-64 relative">
              <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Your location"
                value={location}
                onChange={(e) => setLocation(e.target.value)}
                className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <button
              onClick={() => setShowFilters(!showFilters)}
              className="px-4 py-3 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 flex items-center gap-2"
            >
              <Filter className="w-5 h-5" />
              Filters
            </button>
          </div>

          {/* Filters Panel */}
          {showFilters && (
            <div className="bg-gray-50 border border-gray-200 rounded-lg p-4 mb-4">
              <div className="flex items-center justify-between mb-4">
                <h3 className="font-semibold text-gray-900">Filters</h3>
                <button
                  onClick={() => setShowFilters(false)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>
              <div className="grid grid-cols-4 gap-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Max Distance (miles)
                  </label>
                  <input
                    type="number"
                    value={maxDistance}
                    onChange={(e) => setMaxDistance(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Max Price ($)
                  </label>
                  <input
                    type="number"
                    value={maxPrice}
                    onChange={(e) => setMaxPrice(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Insurance Provider
                  </label>
                  <select
                    value={insurance}
                    onChange={(e) => setInsurance(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">All Insurance</option>
                    <option value="aetna">Aetna</option>
                    <option value="bluecross">Blue Cross Blue Shield</option>
                    <option value="cigna">Cigna</option>
                    <option value="united">UnitedHealthcare</option>
                    <option value="medicare">Medicare</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Availability
                  </label>
                  <select
                    value={availability}
                    onChange={(e) => setAvailability(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="any">Any time</option>
                    <option value="today">Today</option>
                    <option value="week">This week</option>
                    <option value="month">This month</option>
                  </select>
                </div>
              </div>
            </div>
          )}

          {/* View Toggle */}
          <div className="flex items-center gap-2">
            <button
              onClick={() => setViewMode("list")}
              className={`px-4 py-2 rounded-lg ${
                viewMode === "list"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-gray-700 hover:bg-gray-100"
              }`}
            >
              List View
            </button>
            <button
              onClick={() => setViewMode("map")}
              className={`px-4 py-2 rounded-lg ${
                viewMode === "map"
                  ? "bg-blue-600 text-white"
                  : "bg-white text-gray-700 hover:bg-gray-100"
              }`}
            >
              Map View
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 py-6">
        {viewMode === "list" ? (
          <SearchResults onSelectFacility={setSelectedFacility} />
        ) : (
          <MapView onSelectFacility={setSelectedFacility} />
        )}
      </main>

      {/* Facility Modal */}
      {selectedFacility && (
        <FacilityModal
          facility={selectedFacility}
          onClose={() => setSelectedFacility(null)}
        />
      )}
    </div>
  );
}