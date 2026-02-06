import { useState, useEffect } from "react";
import { Search, MapPin, Filter, X } from "lucide-react";
import { SearchResults, UIFacility } from "./components/SearchResults";
import { MapView } from "./components/MapView";
import { FacilityModal } from "./components/FacilityModal";
import { api } from "../lib/api";
import { calculateDistance } from "../lib/utils";
import { Facility } from "../types/api";

export default function App() {
  const [searchQuery, setSearchQuery] = useState("");
  const [location, setLocation] = useState("Lagos, Nigeria"); // Default location
  const [showFilters, setShowFilters] = useState(false);
  const [viewMode, setViewMode] = useState<"list" | "map">("list");
  const [selectedFacility, setSelectedFacility] = useState<any>(null);

  // Data states
  const [facilities, setFacilities] = useState<UIFacility[]>([]);
  const [procedures, setProcedures] = useState<any[]>([]);
  const [insuranceProviders, setInsuranceProviders] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [_error, setError] = useState<string | null>(null);

  // Filter states
  const [selectedProcedure, setSelectedProcedure] = useState("");
  const [maxDistance, setMaxDistance] = useState("50");
  const [maxPrice, setMaxPrice] = useState("500000"); // Updated for NGN
  const [selectedInsurance, setSelectedInsurance] = useState("");
  const [availability, setAvailability] = useState("any");

  // Default search coordinates (Lagos, Nigeria)
  const defaultLat = 6.4531;
  const defaultLon = 3.3958;

  const fetchData = async () => {
    try {
      const [procRes, insRes] = await Promise.all([
        api.getProcedures(),
        api.getInsuranceProviders()
      ]);
      setProcedures(procRes.procedures);
      setInsuranceProviders(insRes.providers);
    } catch (err) {
      console.error("Failed to fetch initial data:", err);
    }
  };

  const fetchFacilities = async () => {
    setLoading(true);
    setError(null);
    try {
      // In a real app, we would geocode the 'location' string to get lat/lon.
      // For now, we use the default coordinates.
      const searchParams = {
        lat: defaultLat,
        lon: defaultLon,
        radius: parseFloat(maxDistance) || 50,
        limit: 50,
        procedure_id: selectedProcedure,
        insurance_provider: selectedInsurance,
      };

      const response = await api.searchFacilities(searchParams);

      const mappedFacilities: UIFacility[] = response.facilities.map((f: Facility) => {
        // ... (rest of mapping)
        const dist = calculateDistance(defaultLat, defaultLon, f.location.latitude, f.location.longitude);
        return {
          id: f.id,
          name: f.name,
          type: f.facility_type || "Health Facility",
          distance: dist,
          price: Math.floor(Math.random() * (800 - 200) + 200), // Mock price
          rating: f.rating || 0,
          reviews: f.review_count || 0,
          nextAvailable: "Today", // Mock
          address: `${f.address.street}, ${f.address.city}, ${f.address.state}`,
          insurance: f.accepted_insurance || [],
          services: ["MRI", "CT Scan", "X-Ray"], // Mock
          urgent: Math.random() > 0.5, // Mock
          capacity: Math.random() > 0.3 ? "Available" : "Limited", // Mock
          waitTime: `${Math.floor(Math.random() * 45 + 5)} min`, // Mock
        };
      });

      setFacilities(mappedFacilities);
    } catch (err) {
      console.error("Failed to fetch facilities:", err);
      setError("Failed to load facilities. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    fetchFacilities();
  }, []); // Fetch on mount

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
              <select
                value={selectedProcedure}
                onChange={(e) => setSelectedProcedure(e.target.value)}
                className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 appearance-none bg-white"
              >
                <option value="">Select a procedure...</option>
                {procedures.map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>
            <div className="w-64 relative">
              <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Your location"
                value={location}
                onChange={(e) => setLocation(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && fetchFacilities()}
                className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <button
              onClick={() => fetchFacilities()}
              className="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 font-semibold"
            >
              Search
            </button>
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
                    value={selectedInsurance}
                    onChange={(e) => setSelectedInsurance(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">All Insurance</option>
                    {insuranceProviders.map(i => (
                      <option key={i.id} value={i.code}>{i.name}</option>
                    ))}
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
          <SearchResults 
            facilities={facilities} 
            loading={loading}
            onSelectFacility={setSelectedFacility} 
          />
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