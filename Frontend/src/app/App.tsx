import { useState, useEffect, useRef } from "react";
import { Search, MapPin, Filter, X, Activity, Navigation } from "lucide-react";
import { SearchResults } from "./components/SearchResults";
import { MapView } from "./components/MapView";
import { FacilityModal } from "./components/FacilityModal";
import { FeedbackTab } from "./components/FeedbackTab";
import { api, API_BASE_URL } from "../lib/api";
import { mapFacilitySearchResultToUI, UIFacility } from "../lib/mappers";
import { FacilitySuggestion } from "../types/api";
import { createRegionalSSEClient, FacilityUpdate, ConnectionStatus } from "../lib/sse-client";

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

  // SSE connection state
  const [sseStatus, setSSEStatus] = useState<ConnectionStatus>("disconnected");
  const [lastSseUpdateAt, setLastSseUpdateAt] = useState<Date | null>(null);
  const sseClientRef = useRef<ReturnType<typeof createRegionalSSEClient> | null>(null);

  // Filter states
  const [maxDistance, setMaxDistance] = useState("50");
  const [maxPrice, setMaxPrice] = useState("500000"); // Updated for NGN
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

  const fetchFacilities = async (overrideQuery?: string, overrideLocation?: string, overrideCenter?: { lat: number; lon: number }) => {
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
      const searchParams = {
        query: queryText || undefined,
        lat: searchCenter.lat,
        lon: searchCenter.lon,
        radius: parseFloat(maxDistance) || 50,
        limit: 50,
        insurance_provider: selectedInsurance || undefined,
        max_price: maxPriceFilter,
      };

      const response = await api.searchFacilities(searchParams);

      const mappedFacilities: UIFacility[] = response.facilities.map((facility) =>
        mapFacilitySearchResultToUI(facility, searchCenter)
      );

      setFacilities(mappedFacilities);
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
           fetchFacilities(undefined, addressString, newCenter);

        } catch (error) {
           console.error("Error getting location details:", error);
           const locString = `${latitude.toFixed(4)}, ${longitude.toFixed(4)}`;
           setLocation(locString);
           fetchFacilities(undefined, locString, newCenter);
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
    fetchFacilities();
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
    fetchFacilities(suggestion.name);
  };

  const handleQuickSearch = (value: string) => {
    setSearchQuery(value);
    setShowSuggestions(false);
    fetchFacilities(value);
  };

  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : currency ? `${currency} ` : "₦";
    return `${symbol}${value.toLocaleString("en-NG")}`;
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b border-gray-200 sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 py-4">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-2xl font-bold text-blue-600">Open Health Initiative</h1>
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 text-xs text-gray-600">
                <Activity className={`w-4 h-4 ${serviceHealth === "ok" ? "text-green-600" : "text-red-500"}`} />
                <span>
                  {serviceHealth === "ok" ? "Service healthy" : "Service issue"}
                </span>
                {serviceCheckedAt && (
                  <span className="text-gray-500">
                    · {serviceCheckedAt.toLocaleTimeString("en-NG", { hour: "2-digit", minute: "2-digit" })}
                  </span>
                )}
              </div>
              {facilities.length > 0 && (
                <div className="flex items-center gap-2 text-xs text-gray-600" title={`Real-time updates: ${sseStatus}`}>
                  <div className={`w-2 h-2 rounded-full ${
                    sseStatus === 'connected' ? 'bg-green-500 animate-pulse' :
                    sseStatus === 'connecting' ? 'bg-yellow-500 animate-pulse' :
                    sseStatus === 'error' ? 'bg-red-500' :
                    'bg-gray-400'
                  }`} />
                  <span className="hidden sm:inline">
                    {sseStatus === 'connected' ? 'Live updates' :
                     sseStatus === 'connecting' ? 'Connecting...' :
                     sseStatus === 'error' ? 'Connection error' :
                     'Offline'}
                  </span>
                </div>
              )}
              <p className="text-sm text-gray-600">Powered by Ateru</p>
            </div>
          </div>
          <div className="mb-4 rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-blue-900">
            Coming soon: AI care agents will call facilities on your behalf to confirm prices and availability.
          </div>

          {/* Search Bar */}
          <div className="flex gap-3 mb-4">
            <div className="flex-1 relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Search hospitals, clinics, procedures..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && fetchFacilities()}
                onFocus={() => {
                  if (suggestions.length > 0) {
                    setShowSuggestions(true);
                  }
                }}
                onBlur={() => {
                  setTimeout(() => setShowSuggestions(false), 150);
                }}
                role="combobox"
                aria-autocomplete="list"
                aria-controls={suggestionListId}
                aria-expanded={showSuggestions}
                aria-haspopup="listbox"
                className="w-full pl-10 pr-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              {showSuggestions && (
                <div
                  className="absolute z-20 mt-2 w-full bg-white border border-gray-200 rounded-lg shadow-lg overflow-hidden"
                  role="listbox"
                  id={suggestionListId}
                  aria-busy={suggestLoading}
                  aria-live="polite"
                >
                  {suggestLoading && (
                    <div className="px-4 py-3 text-sm text-gray-500">
                      Loading suggestions...
                    </div>
                  )}
                  {!suggestLoading && suggestions.length === 0 && (
                    <div className="px-4 py-3 text-sm text-gray-500">
                      No suggestions found.
                    </div>
                  )}
                  {!suggestLoading && suggestions.length > 0 && (
                    <ul>
                      {suggestions.map((suggestion) => (
                        <li key={suggestion.id}>
                          <button
                            type="button"
                            onMouseDown={(e) => e.preventDefault()}
                            onClick={() => handleSuggestionClick(suggestion)}
                            className="w-full text-left px-4 py-3 hover:bg-gray-50"
                            role="option"
                            aria-label={`${suggestion.name}, ${suggestion.address?.city ?? ""}`}
                          >
                            <div className="text-sm font-medium text-gray-900">
                              {suggestion.name}
                            </div>
                            <div className="text-xs text-gray-500">
                              {suggestion.address?.city}
                              {suggestion.address?.state ? `, ${suggestion.address.state}` : ""}
                            </div>
                            {suggestion.matched_service_price ? (
                              <div className="mt-1 text-xs text-gray-600">
                                Price for {suggestion.matched_service_price.name}:{" "}
                                {formatCurrency(
                                  suggestion.matched_service_price.price,
                                  suggestion.matched_service_price.currency
                                )}
                              </div>
                            ) : (
                              suggestion.service_prices &&
                              suggestion.service_prices.length > 0 && (
                                <div className="mt-1 text-xs text-gray-600">
                                  {suggestion.service_prices.slice(0, 2).map((service) => (
                                    <span key={service.procedure_id} className="mr-2">
                                      {service.name}: {formatCurrency(service.price, service.currency)}
                                    </span>
                                  ))}
                                </div>
                              )
                            )}
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </div>
            <div className="w-64 relative">
              <MapPin className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
              <input
                type="text"
                placeholder="Your location"
                value={location}
                onChange={(e) => setLocation(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && fetchFacilities()}
                className="w-full pl-10 pr-12 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={handleUseMyLocation}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-blue-600 hover:text-blue-800"
                title="Use my location"
              >
                <Navigation className="w-5 h-5" />
              </button>
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
          <div className="mb-4 flex flex-wrap items-center gap-2 text-xs text-gray-600">
            <span className="font-medium text-gray-700">Search supports tags:</span>
            <button
              type="button"
              onClick={() => handleQuickSearch("Reliance")}
              className="rounded-full border border-gray-200 bg-white px-3 py-1 text-gray-700 hover:border-blue-300 hover:text-blue-700"
            >
              Reliance
            </button>
            <button
              type="button"
              onClick={() => handleQuickSearch("Ikeja")}
              className="rounded-full border border-gray-200 bg-white px-3 py-1 text-gray-700 hover:border-blue-300 hover:text-blue-700"
            >
              Ikeja
            </button>
            <button
              type="button"
              onClick={() => handleQuickSearch("MRI")}
              className="rounded-full border border-gray-200 bg-white px-3 py-1 text-gray-700 hover:border-blue-300 hover:text-blue-700"
            >
              MRI
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
                    Max Distance (km)
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
                    Max Price (₦)
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
            searchStatus={searchStatus}
            searchDurationMs={searchDurationMs}
            lastSearchAt={lastSearchAt}
            onSelectFacility={setSelectedFacility} 
          />
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

      <div className="border-t border-gray-200 bg-white">
        <div className="max-w-7xl mx-auto px-4 py-3 text-xs text-gray-600 flex items-center gap-2">
          <span className={`h-2 w-2 rounded-full ${
            sseStatus === 'connected' ? 'bg-green-500 animate-pulse' :
            sseStatus === 'connecting' ? 'bg-yellow-500 animate-pulse' :
            sseStatus === 'error' ? 'bg-red-500' :
            'bg-gray-400'
          }`} />
          <span>
            Live updates: {sseStatus === 'connected' ? 'connected' :
            sseStatus === 'connecting' ? 'connecting' :
            sseStatus === 'error' ? 'error' :
            'offline'}
          </span>
          {lastSseUpdateAt && (
            <span>
              · Last SSE update {lastSseUpdateAt.toLocaleTimeString("en-NG", { hour: "2-digit", minute: "2-digit" })}
            </span>
          )}
        </div>
      </div>

      <FeedbackTab />
    </div>
  );
}
