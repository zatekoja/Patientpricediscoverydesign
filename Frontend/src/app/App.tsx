import { useState, useEffect, useRef } from "react";
import { Search, MapPin, Filter, X, Activity, Navigation, List, Map as MapIcon, ChevronDown } from "lucide-react";
import { SearchResults } from "./components/SearchResults";
import { ServiceSuggestionCard } from "./components/ServiceSuggestionCard";
import { MapView } from "./components/MapView";
import { FacilityModal } from "./components/FacilityModal";
import { FeedbackTab } from "./components/FeedbackTab";
import { FAQ } from "./components/FAQ";
import { api, API_BASE_URL } from "../lib/api";
import { useState as useFAQState } from "react";
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
  const [preSelectedService, setPreSelectedService] = useState<string | null>(null);

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
  const [providerHealth, setProviderHealth] = useState<any>(null);
  const [providerHealthLoading, setProviderHealthLoading] = useState(false);
  const [showHealthTooltip, setShowHealthTooltip] = useState(false);
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
    const [minPrice, setMinPrice] = useState(""); // Minimum price filter
  const [selectedInsurance, setSelectedInsurance] = useState("");
  const [availability, setAvailability] = useState("any");
    const [facilityTypes, setFacilityTypes] = useState<string[]>([]);
    const [minRating, setMinRating] = useState("");
  const [showFAQ, setShowFAQ] = useState(false);

    // Available facility types
    const availableFacilityTypes = [
      { value: "hospital", label: "Hospital" },
      { value: "clinic", label: "Clinic" },
      { value: "imaging_center", label: "Imaging Center" },
      { value: "diagnostic_lab", label: "Diagnostic Lab" },
      { value: "pharmacy", label: "Pharmacy" },
      { value: "urgent_care", label: "Urgent Care" },
    ];

  // Default search coordinates (Lagos, Nigeria)
  const defaultLat = 6.5244;
  const defaultLon = 3.3792;
  const [center, setCenter] = useState({ lat: defaultLat, lon: defaultLon });

  const fetchData = async () => {
    try {
      const insRes = await api.getInsuranceProviders();
      setInsuranceProviders(Array.isArray(insRes.providers) ? insRes.providers : []);
    } catch (err) {
      console.error("Failed to fetch initial data:", err);
    }
  };

  const fetchProviderHealth = async () => {
    setProviderHealthLoading(true);
    try {
      const health = await api.getProviderHealth("file_price_list");
      setProviderHealth(health);
    } catch (err) {
      console.error("Failed to fetch provider health:", err);
      setProviderHealth(null);
    } finally {
      setProviderHealthLoading(false);
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
    fetchProviderHealth();
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
          if (update.event_type === 'ward_capacity_update') {
            if (!updated.wardStatuses) updated.wardStatuses = {};
            updated.wardStatuses[fields.ward_id] = {
              status: fields.status,
              count: fields.count,
              trend: fields.trend,
              estimatedWaitMinutes: fields.estimated_wait_minutes,
              lastUpdated: new Date()
            };
            if (fields.avg_wait_minutes) {
              updated.avgWaitMinutes = fields.avg_wait_minutes;
            }
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

  const handleServiceBookNow = (facility: FacilitySuggestion, service: any) => {
    // Map the suggestion facility to UIFacility for the modal
    const uiFacility = mapFacilitySearchResultToUI(
      {
        id: facility.id,
        name: facility.name,
        facility_type: facility.facility_type,
        address: facility.address,
        location: facility.location,
        rating: facility.rating,
        review_count: 0,
        distance_km: 0,
        price: facility.price,
        services: (facility.service_prices || []).map(sp => sp.name),
        service_prices: facility.service_prices || [facility.matched_service_price].filter(Boolean),
        accepted_insurance: [],
        updated_at: new Date().toISOString(),
      } as any,
      center
    );
    
    setPreSelectedService(service.name);
    setSelectedFacility(uiFacility);
    setShowSuggestions(false);
  };

  const handleQuickSearch = (value: string) => {
    setSearchQuery(value);
    setShowSuggestions(false);
    startSearch(value);
  };

  // Group suggestions by service
  const groupServiceSuggestions = (suggestions: FacilitySuggestion[]) => {
    const grouped: Record<string, { service: any; facilities: FacilitySuggestion[] }> = {};

    suggestions.forEach((suggestion) => {
      if (suggestion.matched_service_price) {
        const serviceName = suggestion.matched_service_price.name;
        if (!grouped[serviceName]) {
          grouped[serviceName] = {
            service: {
              name: serviceName,
              description: suggestion.matched_service_price.description,
              procedureId: suggestion.matched_service_price.procedure_id,
              code: suggestion.matched_service_price.code,
              category: suggestion.matched_service_price.category,
              estimatedDuration: suggestion.matched_service_price.estimated_duration,
            },
            facilities: [],
          };
        }
        grouped[serviceName].facilities.push(suggestion);
      } else {
        // Fallback: group by matched_tag or facility name for tag-only matches
        const key = suggestion.matched_tag || suggestion.name;
        if (!grouped[key]) {
          grouped[key] = {
            service: { name: key },
            facilities: [],
          };
        }
        grouped[key].facilities.push(suggestion);
      }
    });

    return Object.values(grouped);
  };

  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : currency ? `${currency} ` : "₦";
    return `${symbol}${value.toLocaleString("en-NG")}`;
  };

  const handleResetFilters = () => {
    setMaxDistance("50");
    setMaxPrice("");
      setMinPrice("");
    setSelectedInsurance("");
    setAvailability("any");
      setFacilityTypes([]);
      setMinRating("");
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
              {/* Consolidated Health Indicator with Tooltip */}
              <div 
                className="relative"
                onMouseEnter={() => setShowHealthTooltip(true)}
                onMouseLeave={() => setShowHealthTooltip(false)}
              >
                <div className="flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium transition-colors cursor-pointer"
                     style={{
                       backgroundColor: serviceHealth === "ok" ? "#f0fdf4" : serviceHealth === "error" ? "#fef2f2" : "#f9fafb",
                       color: serviceHealth === "ok" ? "#15803d" : serviceHealth === "error" ? "#dc2626" : "#6b7280"
                     }}>
                  <Activity className="w-3.5 h-3.5" />
                  <span>
                    {serviceHealth === "ok" ? "All Systems Healthy" : serviceHealth === "error" ? "Service Issues" : "Checking..."}
                  </span>
                  {serviceHealth !== "unknown" && (
                    <div className="w-2 h-2 rounded-full"
                         style={{
                           backgroundColor: serviceHealth === "ok" ? "#15803d" : "#dc2626",
                           animation: serviceHealth === "ok" ? "pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite" : "none"
                         }} />
                  )}
                </div>

                {/* Health Tooltip */}
                {showHealthTooltip && (
                  <div className="absolute top-full right-0 mt-2 w-80 bg-white border border-gray-200 rounded-lg shadow-xl p-4 z-50 animate-in fade-in slide-in-from-top-2 duration-200">
                    <h4 className="text-xs font-bold text-gray-500 uppercase tracking-wide mb-3">System Health Status</h4>
                    <div className="space-y-3">
                      {/* Service Health */}
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          <div className={`w-2 h-2 rounded-full ${
                            serviceHealth === "ok" ? "bg-green-600" : 
                            serviceHealth === "error" ? "bg-red-600" : "bg-gray-400"
                          }`} />
                          <div>
                            <p className="text-sm font-medium text-gray-900">Search API</p>
                            <p className="text-xs text-gray-500">
                              {serviceHealth === "ok" ? "Operational" : 
                               serviceHealth === "error" ? "Down" : "Checking..."}
                            </p>
                          </div>
                        </div>
                        {serviceCheckedAt && (
                          <span className="text-xs text-gray-400">
                            {new Date(serviceCheckedAt).toLocaleTimeString('en-US', { 
                              hour: '2-digit', 
                              minute: '2-digit'
                            })}
                          </span>
                        )}
                      </div>

                      {/* Provider Health */}
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          <div className={`w-2 h-2 rounded-full ${
                            providerHealth?.healthy ? "bg-green-600" : 
                            providerHealth === null ? "bg-gray-400" : "bg-red-600"
                          }`} />
                          <div>
                            <p className="text-sm font-medium text-gray-900">Provider API</p>
                            <p className="text-xs text-gray-500">
                              {providerHealthLoading ? "Checking..." :
                               providerHealth?.healthy ? "Active" : 
                               providerHealth ? "Issues Detected" : "Unavailable"}
                            </p>
                          </div>
                        </div>
                        {providerHealth?.lastSync && (
                          <span className="text-xs text-gray-400">
                            {new Date(providerHealth.lastSync).toLocaleTimeString('en-US', { 
                              hour: '2-digit', 
                              minute: '2-digit'
                            })}
                          </span>
                        )}
                      </div>

                      {/* SSE Connection */}
                      <div className="flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          <div className={`w-2 h-2 rounded-full ${
                            sseStatus === "connected" ? "bg-green-600 animate-pulse" : 
                            sseStatus === "error" ? "bg-red-600" : "bg-gray-400"
                          }`} />
                          <div>
                            <p className="text-sm font-medium text-gray-900">Live Updates (SSE)</p>
                            <p className="text-xs text-gray-500">
                              {sseStatus === "connected" ? "Connected" : 
                               sseStatus === "error" ? "Connection Error" : 
                               sseStatus === "connecting" ? "Connecting..." : "Disconnected"}
                            </p>
                          </div>
                        </div>
                        {lastSseUpdateAt && (
                          <span className="text-xs text-gray-400">
                            {new Date(lastSseUpdateAt).toLocaleTimeString('en-US', { 
                              hour: '2-digit', 
                              minute: '2-digit'
                            })}
                          </span>
                        )}
                      </div>
                    </div>
                    {providerHealth?.message && (
                      <div className="mt-3 pt-3 border-t border-gray-100">
                        <p className="text-xs text-gray-600 italic">{providerHealth.message}</p>
                      </div>
                    )}
                  </div>
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
                  className="absolute z-20 mt-2 w-full bg-white border border-gray-200 rounded-lg shadow-lg overflow-y-auto max-h-96"
                >
                  {!suggestLoading && suggestions.length > 0 ? (
                    <div className="p-3 space-y-3">
                      {groupServiceSuggestions(suggestions).map((group, idx) => (
                        <ServiceSuggestionCard
                          key={`${group.service.name}-${idx}`}
                          service={group.service}
                          facilities={group.facilities}
                          userLat={center.lat}
                          userLon={center.lon}
                          onBookNow={handleServiceBookNow}
                        />
                      ))}
                    </div>
                  ) : suggestLoading ? (
                    <div className="p-4 text-center text-sm text-gray-500">
                      Searching services...
                    </div>
                  ) : (
                    <div className="p-4 text-center text-sm text-gray-500">
                      No services found
                    </div>
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

          {/* Search Latency Display */}
          {searchDurationMs !== null && lastSearchAt && (
            <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
              <div className="flex items-center gap-1.5">
                <Activity className="w-3.5 h-3.5" />
                <span>
                  Search completed in <span className="font-semibold text-gray-700">{searchDurationMs.toFixed(0)}ms</span>
                </span>
              </div>
              <div className="flex items-center gap-1.5">
                <span className="text-gray-400">•</span>
                <span>
                  {new Date(lastSearchAt).toLocaleTimeString('en-US', { 
                    hour: '2-digit', 
                    minute: '2-digit',
                    second: '2-digit'
                  })}
                </span>
              </div>
              {totalCount > 0 && (
                <>
                  <span className="text-gray-400">•</span>
                  <span>
                    <span className="font-semibold text-gray-700">{totalCount.toLocaleString()}</span> results found
                  </span>
                </>
              )}
            </div>
          )}

          {/* Filters Panel */}
          {showFilters && (
            <div className="mt-4 pt-4 border-t border-gray-100 animate-in slide-in-from-top-2 duration-200">
                <div className="space-y-4">
                  {/* First Row */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                  <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Facility Type
                    </label>
                      <div className="space-y-2 max-h-32 overflow-y-auto p-2 border border-gray-200 rounded-lg bg-white">
                        {availableFacilityTypes.map((type) => (
                          <label key={type.value} className="flex items-center gap-2 cursor-pointer hover:bg-gray-50 p-1 rounded">
                            <input
                              type="checkbox"
                              checked={facilityTypes.includes(type.value)}
                              onChange={(e) => {
                                if (e.target.checked) {
                                  setFacilityTypes([...facilityTypes, type.value]);
                                } else {
                                  setFacilityTypes(facilityTypes.filter(t => t !== type.value));
                                }
                              }}
                              className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                            />
                            <span className="text-sm text-gray-700">{type.label}</span>
                          </label>
                        ))}
                      </div>
                  </div>

                  <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Distance (Km)
                    </label>
                    <input
                      type="number"
                        value={maxDistance}
                        onChange={(e) => setMaxDistance(e.target.value)}
                        className="w-full px-3 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        placeholder="Within (Km)"
                        min="0"
                    />
                  </div>

                  <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                      Insurance provider
                    </label>
                    <div className="relative">
                      <select
                        value={selectedInsurance}
                        onChange={(e) => setSelectedInsurance(e.target.value)}
                          className="w-full px-3 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 appearance-none bg-white"
                      >
                          <option value="">Any</option>
                        {(insuranceProviders || []).map(i => (
                          <option key={i.id} value={i.code}>{i.name}</option>
                        ))}
                      </select>
                      <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>

                  <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Minimum Rating
                    </label>
                      <div className="relative">
                      <select
                          value={minRating}
                          onChange={(e) => setMinRating(e.target.value)}
                          className="w-full px-3 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 appearance-none bg-white"
                      >
                          <option value="">Any</option>
                          <option value="4">4+ Stars</option>
                          <option value="3">3+ Stars</option>
                          <option value="2">2+ Stars</option>
                      </select>
                      <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>
                </div>

                  {/* Second Row - Price Range */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Min Price (NGN)
                      </label>
                      <input
                        type="number"
                        value={minPrice}
                        onChange={(e) => setMinPrice(e.target.value)}
                        className="w-full px-3 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        placeholder="From"
                        min="0"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Max Price (NGN)
                      </label>
                      <input
                        type="number"
                        value={maxPrice}
                        onChange={(e) => setMaxPrice(e.target.value)}
                        className="w-full px-3 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        placeholder="To"
                        min="0"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Availability
                      </label>
                      <div className="relative">
                        <select
                          value={availability}
                          onChange={(e) => setAvailability(e.target.value)}
                          className="w-full px-3 py-2.5 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 appearance-none bg-white"
                        >
                          <option value="any">Any time</option>
                          <option value="today">Today</option>
                          <option value="week">This week</option>
                          <option value="month">This month</option>
                        </select>
                        <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                      </div>
                    </div>
                    <div className="flex items-end">
                      <div className="flex items-center gap-3 w-full">
                  <button
                    onClick={handleResetFilters}
                          className="flex-1 px-4 py-2.5 text-sm font-medium text-gray-600 hover:text-gray-900 border border-gray-200 rounded-lg hover:bg-gray-50 bg-white transition-colors"
                  >
                          Reset
                  </button>
                  <button
                    onClick={() => startSearch()}
                          className="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 shadow-sm transition-colors"
                  >
                          Apply
                  </button>
                      </div>
                    </div>
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
          onClose={() => {
            setSelectedFacility(null);
            setPreSelectedService(null);
          }}
          preSelectedServiceName={preSelectedService || undefined}
        />
      )}

      {/* Footer with FAQ */}
      <footer className="border-t border-gray-200 bg-white mt-auto">
        <div className="max-w-7xl mx-auto px-4 py-3 text-xs text-gray-400 flex items-center justify-between">
            <button
              onClick={() => setShowFAQ(true)}
              className="flex items-center gap-2 text-blue-600 hover:text-blue-700 font-medium transition-colors"
            >
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <circle cx="12" cy="12" r="10" strokeWidth="2"/>
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M9.09 9a3 3 0 015.83 1c0 2-3 3-3 3m.08 4h.01"/>
              </svg>
              <span>FAQs</span>
            </button>
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
      <FAQ isOpen={showFAQ} onClose={() => setShowFAQ(false)} />
    </div>
  );
}
