import { useEffect, useState } from "react";
import { MapPin, ArrowRight } from "lucide-react";
import type { FacilitySuggestion } from "../../types/api";
import { api } from "../../lib/api";

// Haversine distance in km
function haversineKm(lat1: number, lon1: number, lat2: number, lon2: number): number {
  const R = 6371;
  const dLat = ((lat2 - lat1) * Math.PI) / 180;
  const dLon = ((lon2 - lon1) * Math.PI) / 180;
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos((lat1 * Math.PI) / 180) *
      Math.cos((lat2 * Math.PI) / 180) *
      Math.sin(dLon / 2) ** 2;
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
}

interface ServiceSuggestionCardProps {
  service: {
    name: string;
    description?: string;
    displayName?: string;
    procedureId?: string;
    code?: string;
    category?: string;
    estimatedDuration?: number;
  };
  facilities: FacilitySuggestion[];
  userLat?: number;
  userLon?: number;
  onBookNow: (facility: FacilitySuggestion, service: any) => void;
}

export function ServiceSuggestionCard({
  service,
  facilities,
  userLat,
  userLon,
  onBookNow,
}: ServiceSuggestionCardProps) {
  if (facilities.length === 0) {
    return null;
  }

  const [enrichedDescription, setEnrichedDescription] = useState<string | null>(null);
  const [enrichmentLoading, setEnrichmentLoading] = useState(false);

  const closestFacility = facilities[0];
  const servicePrice = closestFacility.matched_service_price;
  const description = enrichedDescription || service.description || servicePrice?.description;

  // Try to use the same enrichment description shown in the facility modal.
  // This keeps suggestions helpful even when `procedure.description` is sparse (e.g. "DRESSING").
  useEffect(() => {
    const procedureId = service.procedureId;
    if (!procedureId) return;

    const controller = new AbortController();
    setEnrichmentLoading(true);

    (async () => {
      try {
        const enrichment = await api.getProcedureEnrichment(procedureId);
        if (enrichment?.description) {
          setEnrichedDescription(enrichment.description);
        }
      } catch (err: any) {
        // Ignore aborts; suggestions are transient.
        if (err?.name !== "AbortError") {
          // Keep silent (noisy console in suggestions UX).
        }
      } finally {
        setEnrichmentLoading(false);
      }
    })();

    return () => controller.abort();
  }, [service.procedureId]);
  const distance =
    userLat != null && userLon != null && closestFacility.location
      ? haversineKm(
          userLat,
          userLon,
          closestFacility.location.latitude,
          closestFacility.location.longitude
        )
      : null;

  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : "₦";
    return `${symbol}${value.toLocaleString("en-NG")}`;
  };

  // Show the matched price, and if the facility has a price range show that too
  const priceDisplay = servicePrice
    ? formatCurrency(servicePrice.price, servicePrice.currency)
    : "Price on request";

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-4 hover:shadow-md transition-shadow">
      {/* Header: Service name and Book Now button */}
      <div className="flex items-start justify-between gap-3 mb-3">
        <div className="flex-1 min-w-0">
          <h3 className="text-base font-semibold text-gray-900">{service.displayName || service.name}</h3>
        </div>
        <button
          onClick={() => onBookNow(closestFacility, service)}
          className="flex-shrink-0 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-medium whitespace-nowrap transition-colors flex items-center gap-1.5"
        >
          Book Now
          <ArrowRight className="w-3 h-3" />
        </button>
      </div>

      {/* Closest facility info with distance */}
      <div className="flex items-center gap-2 mb-2 text-sm">
        <MapPin className="w-4 h-4 text-gray-400 flex-shrink-0" />
        <span className="text-gray-700 font-medium truncate">{closestFacility.name}</span>
        <span className="text-gray-500">·</span>
        <span className="text-gray-600 text-xs whitespace-nowrap">
          {distance != null ? `${distance.toFixed(1)} km away` : closestFacility.address?.city || "Nearby"}
        </span>
      </div>

      {/* Price */}
      <div className="mb-2">
        <p className="text-sm font-bold text-gray-900">{priceDisplay}</p>
      </div>

      {/* Description (1-liner) */}
      {description && (
        <p className="text-xs text-gray-600 leading-relaxed line-clamp-2 mb-3">
          {description}
        </p>
      )}

      {!description && enrichmentLoading && (
        <p className="text-xs text-gray-400 leading-relaxed mb-3">Loading description…</p>
      )}

      {/* Additional facilities offering same service */}
      {facilities.length > 1 && (
        <div className="pt-3 border-t border-gray-100">
          <details className="group">
            <summary className="cursor-pointer text-xs font-medium text-blue-600 hover:text-blue-700 flex items-center gap-1">
              <span>View {facilities.length - 1} more location{facilities.length > 2 ? "s" : ""}</span>
              <span className="group-open:rotate-180 transition-transform">▼</span>
            </summary>
            <div className="mt-2 space-y-2">
              {facilities.slice(1).map((facility, idx) => {
                const facPrice = facility.matched_service_price;
                const facPriceDisplay = facPrice
                  ? formatCurrency(facPrice.price, facPrice.currency)
                  : "Price on request";

                return (
                  <div
                    key={idx}
                    className="w-full text-left p-2 hover:bg-gray-50 rounded transition-colors cursor-default"
                  >
                    <div className="flex items-center justify-between gap-2 mb-1">
                      <span className="text-sm font-medium text-gray-900 truncate">
                        {facility.name}
                      </span>
                      <span className="text-xs font-semibold text-gray-900 flex-shrink-0">
                        {facPriceDisplay}
                      </span>
                    </div>
                    <div className="flex items-center gap-2 text-xs text-gray-500">
                      <MapPin className="w-3 h-3 flex-shrink-0" />
                      <span>
                        {facility.address?.city || "Location"}
                      </span>
                    </div>
                  </div>
                );
              })}
            </div>
          </details>
        </div>
      )}
    </div>
  );
}
