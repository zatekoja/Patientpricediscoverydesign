import { MapPin, Calendar, Star, CheckCircle2, Activity, ShieldCheck, Clock, FileText, Phone, Mail, Globe, MessageCircle } from "lucide-react";
import type { UIFacility } from "../../lib/mappers";
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "./ui/pagination";

interface SearchResultsProps {
  facilities: UIFacility[];
  loading: boolean;
  searchStatus?: "idle" | "loading" | "ok" | "error";
  searchDurationMs?: number | null;
  lastSearchAt?: Date | null;
  currentPage: number;
  pageSize: number;
  totalCount?: number;
  onPageChange: (page: number) => void;
  onSelectFacility: (facility: UIFacility) => void;
}

export function SearchResults({
  facilities,
  loading,
  searchStatus = "idle",
  searchDurationMs = null,
  lastSearchAt = null,
  currentPage,
  pageSize,
  totalCount,
  onPageChange,
  onSelectFacility,
}: SearchResultsProps) {

  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : currency ? `${currency} ` : "₦";
    return `${symbol}${value.toLocaleString("en-NG")}`;
  };

  const formatNextAvailable = (iso?: string | null) => {
    if (!iso) return "Check availability";
    const date = new Date(iso);
    if (Number.isNaN(date.getTime())) return "Check availability";
    // Example: "Tomorrow, 09:00am" - simplified for now
    const today = new Date();
    const isToday = date.getDate() === today.getDate() && date.getMonth() === today.getMonth();
    const isTomorrow = new Date(today.getTime() + 86400000).getDate() === date.getDate();
    
    let dayStr = date.toLocaleDateString("en-NG", { weekday: 'short', month: 'short', day: 'numeric' });
    if (isToday) dayStr = "Today";
    if (isTomorrow) dayStr = "Tomorrow";

    const timeStr = date.toLocaleTimeString("en-NG", { hour: "2-digit", minute: "2-digit" });
    return `${dayStr}, ${timeStr}`;
  };

  const resolveCapacity = (status?: string | null) => {
    const normalized = (status || "").toLowerCase();
    if (normalized.includes("available")) {
      return { label: "Available", badge: "bg-green-100 text-green-700", dot: "bg-green-500" };
    }
    if (normalized.includes("limited") || normalized.includes("busy")) {
      return { label: "Limited capacity", badge: "bg-yellow-100 text-yellow-700", dot: "bg-yellow-500" };
    }
    return { label: status || "Unknown", badge: "bg-gray-100 text-gray-700", dot: "bg-gray-400" };
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
        <p className="text-gray-500 text-lg">No facilities found matching your criteria.</p>
      </div>
    );
  }

  const resolvedTotal = totalCount != null
    ? Math.max(totalCount, facilities.length)
    : facilities.length;
  const totalPages = Math.max(1, Math.ceil(resolvedTotal / pageSize));
  
  const buildPaginationItems = (page: number, total: number) => {
    if (total <= 5) {
      return Array.from({ length: total }, (_, index) => index + 1);
    }
    const pages = new Set<number>([1, total, page - 1, page, page + 1]);
    const sorted = Array.from(pages)
      .filter((value) => value >= 1 && value <= total)
      .sort((a, b) => a - b);
    const items: Array<number | "ellipsis"> = [];
    let lastPage = 0;
    sorted.forEach((value) => {
      if (value - lastPage > 1) items.push("ellipsis");
      items.push(value);
      lastPage = value;
    });
    return items;
  };

  const paginationItems = buildPaginationItems(currentPage, totalPages);
  const isPrevDisabled = currentPage <= 1;
  const isNextDisabled = currentPage >= totalPages;

  return (
    <div>
      <div className="space-y-4">
        {facilities.map((facility) => {
          const capacity = resolveCapacity(facility.capacityStatus);
          const priceDisplay = facility.priceMin 
            ? `${formatCurrency(facility.priceMin, facility.currency)}${facility.priceMax && facility.priceMax !== facility.priceMin ? ` - ${formatCurrency(facility.priceMax, facility.currency)}` : ''}`
            : "Price Varies";

          return (
            <div
              key={facility.id}
              className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-md transition-shadow cursor-pointer"
              onClick={() => onSelectFacility(facility)}
            >
              {/* Header Row */}
              <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-2 mb-2">
                <div className="flex items-center gap-3 flex-wrap">
                  <h3 className="text-lg font-semibold text-gray-900">{facility.name}</h3>
                  
                  {/* Urgent Care Badge */}
                  {facility.urgentCareAvailable && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700 border border-blue-100">
                      Urgent care available
                    </span>
                  )}
                  
                  {/* Availability Badge */}
                  <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-medium ${capacity.badge} border border-transparent`}>
                    <span className={`w-1.5 h-1.5 rounded-full ${capacity.dot}`} />
                    {capacity.label}
                  </span>
                </div>
                
                {/* Price Range */}
                <div className="text-right">
                  <span className="block font-semibold text-gray-900">{priceDisplay}</span>
                </div>
              </div>

              {/* Sub-header: Type & Rating */}
              <div className="flex items-center gap-3 mb-4">
                <span className="text-xs font-medium text-gray-500 uppercase tracking-wide">{facility.type}</span>
                <div className="flex items-center gap-1">
                  <Star className="w-3.5 h-3.5 text-yellow-400 fill-yellow-400" />
                  <span className="text-sm font-medium text-gray-900">{facility.rating}</span>
                </div>
              </div>

              {/* Details Row 1 */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 py-3 border-t border-gray-100 border-dashed">
                <div className="flex items-start gap-3">
                   <div className="p-1.5 rounded-full bg-gray-50">
                     <MapPin className="w-4 h-4 text-gray-500" />
                   </div>
                   <div>
                     <p className="text-sm text-gray-900 font-medium">{facility.address}</p>
                     <p className="text-xs text-gray-500">{facility.distanceKm.toFixed(1)} miles away</p>
                   </div>
                </div>
                
                <div className="flex items-start gap-3">
                   <div className="p-1.5 rounded-full bg-gray-50">
                     <Calendar className="w-4 h-4 text-gray-500" />
                   </div>
                   <div>
                     <p className="text-sm text-gray-900 font-medium">{formatNextAvailable(facility.nextAvailableAt)}</p>
                     <p className="text-xs text-gray-500">Next available</p>
                   </div>
                </div>

                <div className="flex items-start gap-3">
                   <div className="p-1.5 rounded-full bg-gray-50">
                     <Clock className="w-4 h-4 text-gray-500" />
                   </div>
                   <div>
                     <p className="text-sm text-gray-900 font-medium">{facility.avgWaitMinutes || '--'} min</p>
                     <p className="text-xs text-gray-500">Avg. wait time</p>
                   </div>
                </div>
              </div>

              {/* Details Row 2 */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pt-3 border-t border-gray-100">
                 <div className="flex items-start gap-3">
                    <div className="mt-0.5">
                      <FileText className="w-4 h-4 text-gray-400" />
                    </div>
                    <div className="text-sm text-gray-600">
                      <span className="text-gray-500 mr-1">Services:</span>
                      {facility.services.slice(0, 3).join(", ")}
                      {facility.services.length > 3 && ", ..."}
                    </div>
                 </div>

                 <div className="flex items-start gap-3">
                    <div className="mt-0.5">
                      <ShieldCheck className="w-4 h-4 text-gray-400" />
                    </div>
                    <div className="text-sm text-gray-600">
                      <span className="text-gray-500 mr-1">Insurances:</span>
                       {facility.insurance.slice(0, 2).join(", ")}
                       {facility.insurance.length > 2 && ", ..."}
                    </div>
                 </div>
              </div>

              {/* Contact Details Row */}
              {(facility.phoneNumber || facility.whatsAppNumber || facility.email || facility.website) && (
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 pt-3 border-t border-gray-100">
                  {facility.phoneNumber && (
                    <div className="flex items-center gap-2">
                      <Phone className="w-4 h-4 text-gray-400" />
                      <a 
                        href={`tel:${facility.phoneNumber}`}
                        className="text-sm text-blue-600 hover:text-blue-800 hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {facility.phoneNumber}
                      </a>
                    </div>
                  )}
                  
                  {(facility.whatsAppNumber || facility.phoneNumber) && (
                    <div className="flex items-center gap-2">
                      <MessageCircle className="w-4 h-4 text-green-500" />
                      <a 
                        href={`https://wa.me/${(facility.whatsAppNumber || facility.phoneNumber)?.replace(/[^0-9]/g, '')}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-green-600 hover:text-green-800 hover:underline"
                        onClick={(e) => e.stopPropagation()}
                      >
                        WhatsApp
                      </a>
                    </div>
                  )}
                  
                  {facility.email && (
                    <div className="flex items-center gap-2">
                      <Mail className="w-4 h-4 text-gray-400" />
                      <a 
                        href={`mailto:${facility.email}`}
                        className="text-sm text-blue-600 hover:text-blue-800 hover:underline truncate"
                        onClick={(e) => e.stopPropagation()}
                      >
                        {facility.email}
                      </a>
                    </div>
                  )}
                  
                  {facility.website && (
                    <div className="flex items-center gap-2">
                      <Globe className="w-4 h-4 text-gray-400" />
                      <a 
                        href={facility.website.startsWith('http') ? facility.website : `https://${facility.website}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-blue-600 hover:text-blue-800 hover:underline truncate"
                        onClick={(e) => e.stopPropagation()}
                      >
                        Visit Website
                      </a>
                    </div>
                  )}
                </div>
              )}

            </div>
          );
        })}
      </div>

      {totalPages > 1 && (
        <div className="mt-8 flex justify-center">
          <Pagination>
            <PaginationContent>
              <PaginationItem>
                <PaginationPrevious
                  href="#"
                  aria-disabled={isPrevDisabled}
                  className={isPrevDisabled ? "pointer-events-none opacity-50" : undefined}
                  onClick={(event) => {
                    event.preventDefault();
                    if (!isPrevDisabled) onPageChange(currentPage - 1);
                  }}
                />
              </PaginationItem>
              {paginationItems.map((item, index) => (
                <PaginationItem key={`${item}-${index}`}>
                  {item === "ellipsis" ? (
                    <PaginationEllipsis />
                  ) : (
                    <PaginationLink
                      href="#"
                      isActive={item === currentPage}
                      onClick={(event) => {
                        event.preventDefault();
                        if (item !== currentPage) onPageChange(item);
                      }}
                    >
                      {item}
                    </PaginationLink>
                  )}
                </PaginationItem>
              ))}
              <PaginationItem>
                <PaginationNext
                  href="#"
                  aria-disabled={isNextDisabled}
                  className={isNextDisabled ? "pointer-events-none opacity-50" : undefined}
                  onClick={(event) => {
                    event.preventDefault();
                    if (!isNextDisabled) onPageChange(currentPage + 1);
                  }}
                />
              </PaginationItem>
            </PaginationContent>
          </Pagination>
        </div>
      )}
    </div>
  );
}
