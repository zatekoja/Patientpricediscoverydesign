import { MapPin, Calendar, Clock, Star, CheckCircle2, Activity } from "lucide-react";
import type { UIFacility } from "../../lib/mappers";

interface SearchResultsProps {
  facilities: UIFacility[];
  loading: boolean;
  searchStatus?: "idle" | "loading" | "ok" | "error";
  searchDurationMs?: number | null;
  lastSearchAt?: Date | null;
  onSelectFacility: (facility: UIFacility) => void;
}

export function SearchResults({
  facilities,
  loading,
  searchStatus = "idle",
  searchDurationMs = null,
  lastSearchAt = null,
  onSelectFacility,
}: SearchResultsProps) {
  const formatUpdate = (iso?: string) => {
    if (!iso) return null;
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return null;
    return date.toLocaleString("en-NG", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const computeNextUpdate = (iso?: string) => {
    if (!iso) return null;
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return null;
    date.setHours(date.getHours() + 24);
    return date.toLocaleString("en-NG", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const statusLabel = {
    idle: "Idle",
    loading: "Searching...",
    ok: "Search OK",
    error: "Search error",
  }[searchStatus];

  const statusColor = {
    idle: "bg-gray-300",
    loading: "bg-blue-500",
    ok: "bg-green-500",
    error: "bg-red-500",
  }[searchStatus];

  const formatNextAvailable = (iso?: string | null) => {
    if (!iso) return "Check availability";
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return "Check availability";
    return date.toLocaleString("en-NG", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const resolveCapacity = (status?: string | null) => {
    const normalized = (status || "").toLowerCase();
    if (normalized.includes("available")) {
      return { label: "Available", tone: "green", badge: "bg-green-100 text-green-800", dot: "bg-green-600" };
    }
    if (normalized.includes("limited") || normalized.includes("busy")) {
      return { label: status || "Limited", tone: "yellow", badge: "bg-yellow-100 text-yellow-800", dot: "bg-yellow-600" };
    }
    if (normalized.includes("full") || normalized.includes("closed")) {
      return { label: status || "Full", tone: "red", badge: "bg-red-100 text-red-800", dot: "bg-red-600" };
    }
    return { label: status || "Unknown", tone: "gray", badge: "bg-gray-100 text-gray-700", dot: "bg-gray-400" };
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  if (facilities.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500 text-lg">No facilities found.</p>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">
            {facilities.length} facilities found
          </h2>
          <p className="text-sm text-gray-600 mt-1">
            Showing results near your location
          </p>
          <div className="mt-2 text-xs text-gray-500">
            Transparency: facility data refreshes every 24 hours.
          </div>
        </div>
        <div className="flex flex-col items-end gap-2">
          <div className="flex items-center gap-2 text-xs text-gray-600">
            <span className={`h-2 w-2 rounded-full ${statusColor}`} />
            <span>{statusLabel}</span>
            {searchDurationMs != null && (
              <span>· {Math.round(searchDurationMs)} ms</span>
            )}
            {lastSearchAt && (
              <span>
                · {lastSearchAt.toLocaleTimeString("en-NG", { hour: "2-digit", minute: "2-digit" })}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <label className="text-sm text-gray-700">Sort by:</label>
            <select className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option>Distance</option>
              <option>Price (Low to High)</option>
              <option>Price (High to Low)</option>
              <option>Rating</option>
              <option>Availability</option>
            </select>
          </div>
        </div>
      </div>

      <div className="space-y-4">
        {facilities.map((facility) => (
          <div
            key={facility.id}
            className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-lg transition-shadow cursor-pointer"
            onClick={() => onSelectFacility(facility)}
          >
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div className="flex-1">
                <div className="flex items-start gap-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="text-lg font-semibold text-gray-900">
                        {facility.name}
                      </h3>
                      {facility.urgentCareAvailable && (
                        <span className="px-2 py-1 bg-red-100 text-red-700 text-xs rounded-full">
                          Urgent Care Available
                        </span>
                      )}
                    </div>
                    <p className="text-sm text-gray-600 mb-2">{facility.type}</p>
                    <div className="flex items-center gap-1 mb-3">
                      <Star className="w-4 h-4 text-yellow-400 fill-yellow-400" />
                      <span className="font-semibold text-gray-900">
                        {facility.rating}
                      </span>
                      <span className="text-sm text-gray-600">
                        ({facility.reviews} reviews)
                      </span>
                    </div>
                  </div>
                </div>

                {/* Key Information Grid */}
                <div className="grid grid-cols-2 gap-4 mb-4 lg:grid-cols-3">
                  <div className="flex items-start gap-2">
                    <MapPin className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {facility.distanceKm.toFixed(2)} km away
                      </p>
                      <p className="text-xs text-gray-600">{facility.address}</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <Calendar className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {formatNextAvailable(facility.nextAvailableAt)}
                      </p>
                      <p className="text-xs text-gray-600">Next available</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <Activity className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      {(() => {
                        const capacity = resolveCapacity(facility.capacityStatus);
                        return (
                          <>
                            <div className={`inline-flex items-center gap-2 rounded-full px-2 py-1 text-xs ${capacity.badge}`}>
                              <span className={`h-2 w-2 rounded-full ${capacity.dot}`} />
                              <span className="font-medium">Capacity {capacity.label}</span>
                            </div>
                            <p className="text-xs text-gray-600 mt-1">
                              {facility.avgWaitMinutes != null
                                ? `Avg. wait ${facility.avgWaitMinutes} min`
                                : "Avg. wait not available"}
                            </p>
                          </>
                        );
                      })()}
                    </div>
                  </div>
                </div>

                <div className="text-xs text-gray-500 mb-4">
                  {formatUpdate(facility.updatedAt) && (
                    <span>
                      Last updated: {formatUpdate(facility.updatedAt)}
                    </span>
                  )}
                  {computeNextUpdate(facility.updatedAt) && (
                    <span>
                      {" "}· Next update: {computeNextUpdate(facility.updatedAt)}
                    </span>
                  )}
                </div>

                {/* Services & Insurance */}
                <div className="space-y-2">
                  {facility.services.length > 0 && (
                    <div className="flex items-center gap-2">
                      <CheckCircle2 className="w-4 h-4 text-green-600" />
                      <span className="text-sm text-gray-700">
                        Services: {facility.services.join(", ")}
                      </span>
                    </div>
                  )}
                  {facility.insurance.length > 0 && (
                    <div className="flex items-center gap-2">
                      <CheckCircle2 className="w-4 h-4 text-green-600" />
                      <span className="text-sm text-gray-700">
                        Insurance: {facility.insurance.slice(0, 3).join(", ")}
                        {facility.insurance.length > 3 && ` +${facility.insurance.length - 3} more`}
                      </span>
                    </div>
                  )}
                </div>
              </div>

              {/* Right Side - Capacity Status */}
              <div className="flex items-center justify-between lg:flex-col lg:items-end lg:text-right">
                <button className="px-6 py-2 bg-transparent text-blue-600 border border-blue-600 rounded-lg hover:bg-blue-50 transition-colors">
                  View Details
                </button>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
