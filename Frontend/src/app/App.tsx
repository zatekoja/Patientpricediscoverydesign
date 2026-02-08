import { useState, useEffect, useRef } from "react";
import { Search, MapPin, Filter, X, Activity, Navigation, List, Map as MapIcon, ChevronDown } from "lucide-react";
import { SearchResults } from "./components/SearchResults";
import { MapView } from "./components/MapView";
import { FacilityModal } from "./components/FacilityModal";
import { FeedbackTab } from "./components/FeedbackTab";
import { api, API_BASE_URL } from "../lib/api";
import { mapFacilitySearchResultToUI, UIFacility } from "../lib/mappers";
import { FacilitySuggestion } from "../types/api";
import { createRegionalSSEClient, FacilityUpdate, ConnectionStatus } from "../lib/sse-client";
import logo from "../assets/logo.png";

export default function App() {
  const suggestionListId = "facility-suggestions";
  const [searchQuery, setSearchQuery] = useState("");
  const [location, setLocation] = useState("Lagos, Nigeria"); // Default location
  const [showFilters, setShowFilters] = useState(false);
  const [viewMode, setViewMode] = useState<"list" | "map">("list");
  const [selectedFacility, setSelectedFacility] = useState<UIFacility | null>(null);

  // Data states
  const [facilities, setFacilities] = useState<UIFacility[]>([]);
  const [insuranceProviders, setInsuranceProviders] = useState<any[]>([]);
  const [suggestions, setSuggestions] = useState<FacilitySuggestion[]>([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [suggestLoading, setSuggestLoading] = useState(false);
  const [loading, setLoading] = useState(false);
  const [_error, setError] = useState<string | null>(null);
  const [searchStatus, setSearchStatus] = useState<"idle" | "loading" | "ok" | "error">("idle");
  const [searchDurationMs, setSearchDurationMs] = useState<number | null>(null);
  const [lastSearchAt, setLastSearchAt] = useState<Date | null>(null);
  const [serviceHealth, setServiceHealth] = useState<"unknown" | "ok" | "error">("unknown");
  const [serviceCheckedAt, setServiceCheckedAt] = useState<Date | null>(null);
  const [totalCount, setTotalCount] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const pageSize = 50;

  // SSE connection state
  const [sseStatus, setSSEStatus] = useState<ConnectionStatus>("disconnected");
  const [lastSseUpdateAt, setLastSseUpdateAt] = useState<Date | null>(null);
  const sseClientRef = useRef<ReturnType<typeof createRegionalSSEClient> | null>(null);

  // Filter states
  const [maxDistance, setMaxDistance] = useState("50");
  const [maxPrice, setMaxPrice] = useState(""); // Optional filter; leave empty to include all
  const [selectedInsurance, setSelectedInsurance] = useState("");
  const [availability, setAvailability] = useState("any");

  // Default search coordinates (Lagos, Nigeria)
  const defaultLat = 6.5244;
  const defaultLon = 3.3792;
  const [center, setCenter] = useState({ lat: defaultLat, lon: defaultLon });

  const fetchData = async () => {
    try {
      const insRes = await api.getInsuranceProviders();
      setInsuranceProviders(insRes.providers);
    } catch (err) {
      console.error("Failed to fetch initial data:", err);
    }
  };

  const fetchFacilities = async (
    overrideQuery?: string,
    overrideLocation?: string,
    overrideCenter?: { lat: number; lon: number },
    overridePage?: number
  ) => {
    setLoading(true);
    setError(null);
    setSearchStatus("loading");
    const startTime = performance.now();
    try {
      let searchCenter = overrideCenter || center;
      const locationInput = overrideLocation ?? location;
      const trimmedLocation = locationInput.trim();

      // Only geocode if we don't have an override center and there is a location string
      if (!overrideCenter && trimmedLocation.length > 0) {
        try {
          const geo = await api.geocode(trimmedLocation);
          searchCenter = { lat: geo.lat, lon: geo.lon };
          setCenter(searchCenter);
        } catch (geoErr) {
          console.error("Failed to geocode location, using previous center:", geoErr);
        }
      } else if (overrideCenter) {
           // Ensure center state is updated when override provided
           setCenter(overrideCenter);
      }

      const queryText = (overrideQuery ?? searchQuery).trim();
      const maxPriceValue = parseFloat(maxPrice);
      const maxPriceFilter = Number.isFinite(maxPriceValue) ? maxPriceValue : undefined;
      const page = overridePage ?? currentPage;
      const offset = (page - 1) * pageSize;
      const searchParams = {
        query: queryText || undefined,
        lat: searchCenter.lat,
        lon: searchCenter.lon,
        radius: parseFloat(maxDistance) || 50,
        limit: pageSize,
        offset,
        insurance_provider: selectedInsurance || undefined,
        max_price: maxPriceFilter,
      };

      console.log("[search] /facilities/search", {
        baseUrl: API_BASE_URL,
        params: searchParams,
        page,
      });

      const response = await api.searchFacilities(searchParams);

      const mappedFacilities: UIFacility[] = response.facilities.map((facility) =>
        mapFacilitySearchResultToUI(facility, searchCenter)
      );

      setFacilities(mappedFacilities);
      setTotalCount(Number.isFinite(response.count) ? response.count : mappedFacilities.length);
      setCurrentPage(page);
      setSearchStatus("ok");
    } catch (err) {
      console.error("Failed to fetch facilities:", err);
      setError("Failed to load facilities. Please try again.");
      setSearchStatus("error");
    } finally {
      setLoading(false);
      setSearchDurationMs(performance.now() - startTime);
      setLastSearchAt(new Date());
    }
  };

  const startSearch = (
    overrideQuery?: string,
    overrideLocation?: string,
    overrideCenter?: { lat: number; lon: number }
  ) => {
    const firstPage = 1;
    setCurrentPage(firstPage);
    fetchFacilities(overrideQuery, overrideLocation, overrideCenter, firstPage);
  };

  const handlePageChange = (page: number) => {
    const fallbackTotal = totalCount || facilities.length;
    const maxPage = Math.max(1, Math.ceil(fallbackTotal / pageSize));
    const nextPage = Math.min(Math.max(page, 1), maxPage);
    if (nextPage === currentPage) return;
    fetchFacilities(undefined, undefined, undefined, nextPage);
  };

  const handleUseMyLocation = () => {
    if (!navigator.geolocation) {
      alert("Geolocation is not supported by your browser");
      return;
    }

    setLoading(true);
    navigator.geolocation.getCurrentPosition(
      async (position) => {
        const { latitude, longitude } = position.coords;
        const newCenter = { lat: latitude, lon: longitude };

        try {
           // Reverse geocode to get address for text input
           const addressData = await api.reverseGeocode(latitude, longitude);
           const addressString = addressData.FormattedAddress || `${latitude.toFixed(4)}, ${longitude.toFixed(4)}`;

           setLocation(addressString);
           startSearch(undefined, addressString, newCenter);

        } catch (error) {
           console.error("Error getting location details:", error);
           const locString = `${latitude.toFixed(4)}, ${longitude.toFixed(4)}`;
           setLocation(locString);
           startSearch(undefined, locString, newCenter);
        }
      },
      (error) => {
        console.error("Error getting location:", error);
        setLoading(false);
        alert("Unable to retrieve your location");
      }
    );
  };

  useEffect(() => {
    fetchData();
    fetchFacilities(undefined, undefined, undefined, 1);
  }, []); // Fetch on mount

  useEffect(() => {
    const checkHealth = async () => {
      const baseUrl = API_BASE_URL.replace(/\/api$/, "");
      try {
        const res = await fetch(`${baseUrl}/health`);
        setServiceHealth(res.ok ? "ok" : "error");
      } catch {
        setServiceHealth("error");
      } finally {
        setServiceCheckedAt(new Date());
      }
    };
    checkHealth();
  }, []);

  useEffect(() => {
    const query = searchQuery.trim();
    if (query.length < 2) {
      setSuggestions([]);
      setShowSuggestions(false);
      return;
    }

    const controller = new AbortController();
    const timer = setTimeout(async () => {
      setSuggestLoading(true);
      try {
        const response = await api.suggestFacilities(
          { query, lat: center.lat, lon: center.lon, limit: 6 },
          controller.signal
        );
        setSuggestions(response.suggestions);
        setShowSuggestions(true);
      } catch (err: any) {
        if (err?.name !== "AbortError") {
          console.error("Failed to fetch suggestions:", err);
        }
      } finally {
        setSuggestLoading(false);
      }
    }, 250);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [searchQuery, center.lat, center.lon]);

  // SSE connection for real-time updates
  useEffect(() => {
    // Only connect when we have facilities to monitor
    if (facilities.length === 0) {
      return;
    }

    const radiusKm = parseInt(maxDistance) || 50;
    const client = createRegionalSSEClient(center.lat, center.lon, radiusKm);
    sseClientRef.current = client;

    // Subscribe to status changes
    const unsubscribeStatus = client.onStatusChange((status) => {
      setSSEStatus(status);
      console.log('SSE status changed:', status);
    });

    // Subscribe to facility updates
    const unsubscribeUpdates = client.onUpdate((update: FacilityUpdate) => {
      console.log('Received facility update:', update);
      handleFacilityUpdate(update);
    });

    // Connect to the stream
    client.connect();

    // Cleanup on unmount or when dependencies change
    return () => {
      unsubscribeStatus();
      unsubscribeUpdates();
      client.disconnect();
      sseClientRef.current = null;
    };
  }, [center.lat, center.lon, maxDistance, facilities.length]);

  // Handler for real-time facility updates
  const handleFacilityUpdate = (update: FacilityUpdate) => {
    if (update.timestamp) {
      const parsed = new Date(update.timestamp);
      if (!Number.isNaN(parsed.getTime())) {
        setLastSseUpdateAt(parsed);
      } else {
        setLastSseUpdateAt(new Date());
      }
    } else {
      setLastSseUpdateAt(new Date());
    }

    setFacilities(prevFacilities => {
      return prevFacilities.map(facility => {
        if (facility.id === update.facility_id) {
          // Update the facility with new values
          const updated = { ...facility };
          const fields = update.changed_fields ?? {};

          if ('capacity_status' in fields) {
            updated.capacityStatus = fields.capacity_status;
          }
          if ('avg_wait_minutes' in fields) {
            updated.avgWaitMinutes = fields.avg_wait_minutes;
          }
          if ('urgent_care_available' in fields) {
            updated.urgentCareAvailable = fields.urgent_care_available;
          }

          if (update.event_type === 'service_availability_update') {
            const procedureId = typeof fields.procedure_id === 'string' ? fields.procedure_id : undefined;
            const procedureName = typeof fields.procedure_name === 'string' ? fields.procedure_name : undefined;
            const isAvailable = typeof fields.is_available === 'boolean' ? fields.is_available : undefined;

            if (typeof isAvailable === 'boolean') {
              if (isAvailable) {
                if (procedureName && !updated.services.includes(procedureName)) {
                  updated.services = [...updated.services, procedureName];
                }

                const alreadyExists = updated.servicePrices.some((service) => {
                  if (procedureId && service.procedureId === procedureId) return true;
                  return procedureName ? service.name === procedureName : false;
                });

                if (!alreadyExists && procedureName) {
                  updated.servicePrices = [
                    ...updated.servicePrices,
                    {
                      procedureId,
                      name: procedureName,
                      price: typeof fields.price === 'number' ? fields.price : 0,
                      currency: typeof fields.currency === 'string' ? fields.currency : (updated.currency || 'NGN'),
                      description: typeof fields.procedure_description === 'string' ? fields.procedure_description : undefined,
                      category: typeof fields.procedure_category === 'string' ? fields.procedure_category : undefined,
                      code: typeof fields.procedure_code === 'string' ? fields.procedure_code : undefined,
                      estimatedDuration: typeof fields.estimated_duration === 'number' ? fields.estimated_duration : undefined,
                    },
                  ];
                }
              } else {
                if (procedureName) {
                  updated.services = updated.services.filter((service) => service !== procedureName);
                }

                if (procedureId || procedureName) {
                  updated.servicePrices = updated.servicePrices.filter((service) => {
                    if (procedureId && service.procedureId === procedureId) return false;
                    if (procedureName && service.name === procedureName) return false;
                    return true;
                  });
                }
              }
            }
          }

          console.log(`Updated facility ${facility.name}:`, updated);
          return updated;
        }
        return facility;
      });
    });
  };

  const handleSuggestionClick = (suggestion: FacilitySuggestion) => {
    setSearchQuery(suggestion.name);
    setShowSuggestions(false);
    startSearch(suggestion.name);
  };

  const handleQuickSearch = (value: string) => {
    setSearchQuery(value);
    setShowSuggestions(false);
    startSearch(value);
  };

  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : currency ? `${currency} ` : "₦";
    return `${symbol}${value.toLocaleString("en-NG")}`;
  };

  const handleResetFilters = () => {
    setMaxDistance("50");
    setMaxPrice("");
    setSelectedInsurance("");
    setAvailability("any");
  };

  return (
    <div className="min-h-screen bg-white flex flex-col font-sans">
      {/* Header */}
      <header className="bg-white py-4">
        <div className="max-w-[1440px] mx-auto px-4 md:px-6 lg:px-8">
          <div className="flex items-center justify-between mb-8">
            <div className="flex items-center gap-2">
               <img src={logo} alt="Open Health Initiative" className="h-10 md:h-12 w-auto" />
            </div>
            <div className="flex items-center gap-4">
              {/* Service Health Indicator */}
              <div className="flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium transition-colors"
                   style={{
                     backgroundColor: serviceHealth === "ok" ? "#f0fdf4" : serviceHealth === "error" ? "#fef2f2" : "#f9fafb",
                     color: serviceHealth === "ok" ? "#15803d" : serviceHealth === "error" ? "#dc2626" : "#6b7280"
                   }}>
                <Activity className="w-3.5 h-3.5" />
                <span>
                  {serviceHealth === "ok" ? "Service Healthy" : serviceHealth === "error" ? "Service Issues" : "Checking..."}
                </span>
                {serviceHealth !== "unknown" && (
                  <div className="w-2 h-2 rounded-full"
                       style={{
                         backgroundColor: serviceHealth === "ok" ? "#15803d" : "#dc2626",
                         animation: serviceHealth === "ok" ? "pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite" : "none"
                       }} />
                )}
              </div>
              <div className="text-right">
                <p className="text-xs text-gray-400 uppercase tracking-wide">Powered by Ateru</p>
              </div>
            </div>
          </div>

          {/* Search Bar & Controls */}
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center">
            {/* View Toggle */}
            <div className="bg-gray-100 p-1 rounded-lg flex items-center gap-1 w-fit">
              <button
                onClick={() => setViewMode("list")}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  viewMode === "list"
                    ? "bg-blue-600 text-white shadow-sm"
                    : "text-gray-600 hover:text-gray-900"
                }`}
              >
                List view
              </button>
              <button
                onClick={() => setViewMode("map")}
                className={`px-4 py-2 rounded-md text-sm font-medium transition-colors ${
                  viewMode === "map"
                    ? "bg-blue-600 text-white shadow-sm"
                    : "text-gray-600 hover:text-gray-900"
                }`}
              >
                Map view
              </button>
            </div>

            {/* Search Inputs */}
            <div className="flex-1 flex flex-col md:flex-row gap-4">
              <div className="flex-1 relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
                <input
                  type="text"
                  placeholder="Search for procedures (e.g MRI, ct scan, X-rays)"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && startSearch()}
                  onFocus={() => {
                    if (suggestions.length > 0) {
                      setShowSuggestions(true);
                    }
                  }}
                  onBlur={() => {
                    setTimeout(() => setShowSuggestions(false), 150);
                  }}
                  className="w-full pl-10 pr-4 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                />
                 {showSuggestions && (
                <div
                  className="absolute z-20 mt-2 w-full bg-white border border-gray-200 rounded-lg shadow-lg overflow-hidden"
                >
                  {/* ... Suggestions rendering same as before ... */}
                   {!suggestLoading && suggestions.length > 0 && (
                    <ul>
                      {suggestions.map((suggestion) => (
                        <li key={suggestion.id}>
                          <button
                            type="button"
                            onMouseDown={(e) => e.preventDefault()}
                            onClick={() => handleSuggestionClick(suggestion)}
                            className="w-full text-left px-4 py-3 hover:bg-gray-50"
                          >
                            <div className="text-sm font-medium text-gray-900">
                              {suggestion.name}
                            </div>
                            <div className="text-xs text-gray-500">
                               {suggestion.address?.city}
                            </div>
                          </button>
                        </li>
                      ))}
                    </ul>
                   )}
                </div>
              )}
              </div>
              <div className="w-full md:w-80 relative">
                <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
                <input
                  type="text"
                  placeholder="Enter your location"
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && startSearch()}
                  className="w-full pl-10 pr-10 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                />
                 <button
                  onClick={handleUseMyLocation}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-blue-600"
                  title="Use my location"
                >
                  <Navigation className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Filter Button */}
            <button
              onClick={() => setShowFilters(!showFilters)}
              className={`px-4 py-2.5 text-sm border rounded-lg font-medium flex items-center gap-2 transition-colors ${
                showFilters 
                ? "bg-blue-50 border-blue-200 text-blue-700"
                : "bg-white border-gray-200 text-gray-700 hover:bg-gray-50"
              }`}
            >
              <Filter className="w-4 h-4" />
              Filter
            </button>
          </div>

          {/* Filters Panel */}
          {showFilters && (
            <div className="mt-4 pt-4 border-t border-gray-100 animate-in slide-in-from-top-2 duration-200">
              <div className="flex flex-col lg:flex-row gap-4 items-end">
                <div className="flex-1 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 w-full">
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1.5">
                      Max distance (Km)
                    </label>
                    <input
                      type="number"
                      value={maxDistance}
                      onChange={(e) => setMaxDistance(e.target.value)}
                      className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="0"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1.5">
                      Max price (In naira)
                    </label>
                    <input
                      type="number"
                      value={maxPrice}
                      onChange={(e) => setMaxPrice(e.target.value)}
                      className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="NGN"
                    />
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1.5">
                      Insurance provider
                    </label>
                    <div className="relative">
                      <select
                        value={selectedInsurance}
                        onChange={(e) => setSelectedInsurance(e.target.value)}
                        className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 appearance-none bg-white"
                      >
                        <option value="">Select</option>
                        {insuranceProviders.map(i => (
                          <option key={i.id} value={i.code}>{i.name}</option>
                        ))}
                      </select>
                      <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>
                  <div>
                    <label className="block text-xs font-medium text-gray-700 mb-1.5">
                      Availability
                    </label>
                     <div className="relative">
                      <select
                        value={availability}
                        onChange={(e) => setAvailability(e.target.value)}
                        className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 appearance-none bg-white"
                      >
                        <option value="any">Select</option>
                        <option value="today">Today</option>
                        <option value="week">This week</option>
                        <option value="month">This month</option>
                      </select>
                      <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3 w-full lg:w-auto mt-2 lg:mt-0">
                  <button
                    onClick={handleResetFilters}
                    className="px-4 py-2 text-sm font-medium text-gray-600 hover:text-gray-900 border border-gray-200 rounded-lg hover:bg-gray-50 bg-white"
                  >
                    Reset changes
                  </button>
                  <button
                    onClick={() => startSearch()}
                    className="px-6 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 shadow-sm"
                  >
                    Apply filters
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-[1440px] mx-auto px-4 md:px-6 lg:px-8 py-6 flex-1 w-full">
        {viewMode === "list" ? (
          <div className="bg-transparent">
            <SearchResults 
              facilities={facilities} 
              loading={loading}
              searchStatus={searchStatus}
              searchDurationMs={searchDurationMs}
              lastSearchAt={lastSearchAt}
              currentPage={currentPage}
              pageSize={pageSize}
              totalCount={totalCount}
              onPageChange={handlePageChange}
              onSelectFacility={setSelectedFacility}
            />
          </div>
        ) : (
          <MapView
            facilities={facilities}
            center={center}
            onSelectFacility={setSelectedFacility}
          />
        )}
      </main>

      {/* Facility Modal */}
      {selectedFacility && (
        <FacilityModal
          facility={selectedFacility}
          onClose={() => setSelectedFacility(null)}
        />
      )}

      {/* Footer Status Line */}
      <footer className="border-t border-gray-200 bg-white mt-auto">
         {/* Kept minimal footer for debug info */}
        <div className="max-w-7xl mx-auto px-4 py-3 text-xs text-gray-400 flex items-center justify-between">
            <div className="flex items-center gap-2">
                 <span className={`h-2 w-2 rounded-full ${
                    serviceHealth === "ok" ? "bg-green-500" : "bg-red-500"
                 }`} />
                 <span>System status: {serviceHealth}</span>
            </div>
            <div className="flex items-center gap-2">
                <span className={`h-2 w-2 rounded-full ${
                  sseStatus === 'connected' ? 'bg-green-500' :
                  sseStatus === 'connecting' ? 'bg-yellow-500' :
                  'bg-gray-400'
                }`} />
                <span>Real-time updates: {sseStatus}</span>
            </div>
        </div>
      </footer>

      <FeedbackTab />
    </div>
  );
}
