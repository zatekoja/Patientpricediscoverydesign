import { MapPin, Calendar, Star, CheckCircle2, Activity, ShieldCheck, Clock, FileText, Phone, Mail, Globe, MessageCircle } from "lucide-react";
import { useState } from "react";
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

type FacilityCategory = "all" | "hospitals" | "laboratories" | "pharmacies" | "sti_testing";

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
  const [activeCategory, setActiveCategory] = useState<FacilityCategory>("all");

  const categories = [
    { id: "all" as FacilityCategory, label: "All Facilities" },
    { id: "hospitals" as FacilityCategory, label: "Hospitals & Clinics" },
    { id: "laboratories" as FacilityCategory, label: "Laboratories" },
    { id: "pharmacies" as FacilityCategory, label: "Pharmacies" },
    { id: "sti_testing" as FacilityCategory, label: "STI Testing" },
  ];

  const filterFacilitiesByCategory = (facilities: UIFacility[], category: FacilityCategory): UIFacility[] => {
    if (category === "all") return facilities;
    
    return facilities.filter(facility => {
      const type = facility.type?.toLowerCase() || "";
      const services = facility.services.map(s => s.toLowerCase());
      
      switch (category) {
        case "hospitals":
          return type.includes("hospital") || type.includes("clinic") || type.includes("center");
        case "laboratories":
          return type.includes("lab") || type.includes("diagnostic") || 
                 services.some(s => s.includes("lab") || s.includes("blood test") || s.includes("x-ray"));
        case "pharmacies":
          return type.includes("pharmacy") || services.some(s => s.includes("pharmacy") || s.includes("medication"));
        case "sti_testing":
          return services.some(s => s.includes("sti") || s.includes("std") || s.includes("hiv") || s.includes("sexual health"));
        default:
          return true;
      }
    });
  };

  const filteredFacilities = filterFacilitiesByCategory(facilities, activeCategory);

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

  const resolvedTotal = totalCount != null
    ? Math.max(totalCount, filteredFacilities.length)
    : filteredFacilities.length;
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
      {/* Category Tabs */}
      <div className="mb-6 border-b border-gray-200">
        <div className="flex gap-1 overflow-x-auto">
          {categories.map((category) => (
            <button
              key={category.id}
              onClick={() => setActiveCategory(category.id)}
              className={`px-6 py-3 text-sm font-medium whitespace-nowrap transition-colors border-b-2 ${
                activeCategory === category.id
                  ? "text-blue-600 border-blue-600"
                  : "text-gray-600 border-transparent hover:text-gray-900 hover:border-gray-300"
              }`}
            >
              {category.label}
            </button>
          ))}
        </div>
      </div>

      {/* No results message */}
      {filteredFacilities.length === 0 && !loading && (
        <div className="text-center py-12">
          <p className="text-gray-500 text-lg">No facilities found matching your criteria.</p>
        </div>
      )}

      <div className="space-y-4">
        {filteredFacilities.map((facility) => {
          const capacity = resolveCapacity(facility.capacityStatus);
          const priceDisplay = facility.priceMin 
            ? `N${facility.priceMin.toLocaleString()}${facility.priceMax && facility.priceMax !== facility.priceMin ? ` - N${facility.priceMax.toLocaleString()}` : ''}`
            : "Price Varies";

          return (
            <div
              key={facility.id}
              className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-lg transition-shadow cursor-pointer"
              onClick={() => onSelectFacility(facility)}
            >
              {/* Header Row */}
              <div className="flex items-start justify-between mb-3">
                <div className="flex-1">
                  <div className="flex items-center gap-3 flex-wrap mb-2">
                    <h3 className="text-xl font-semibold text-gray-900">{facility.name}</h3>
                    
                    {/* Urgent Care Badge */}
                    {facility.urgentCareAvailable && (
                      <span className="inline-flex items-center px-3 py-1 rounded-md text-xs font-medium bg-blue-50 text-blue-700">
                        Urgent care available
                      </span>
                    )}
                    
                    {/* Availability Badge */}
                    <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium ${
                      capacity.label === "Available" ? "bg-green-50 text-green-700" :
                      capacity.label === "Limited capacity" ? "bg-yellow-50 text-yellow-700" :
                      "bg-gray-50 text-gray-700"
                    }`}>
                      <span className={`w-2 h-2 rounded-full ${capacity.dot}`} />
                      {capacity.label}
                    </span>
                  </div>
                  
                  {/* Type & Rating */}
                  <div className="flex items-center gap-3">
                    <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">{facility.type}</span>
                    <div className="flex items-center gap-1">
                      <Star className="w-4 h-4 text-yellow-400 fill-yellow-400" />
                      <span className="text-sm font-semibold text-gray-900">{facility.rating}</span>
                    </div>
                  </div>
                </div>
                
                {/* Price Range */}
                <div className="text-right ml-4">
                  <span className="text-xl font-bold text-gray-900">{priceDisplay}</span>
                </div>
              </div>

              {/* Details Grid */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-6 py-4 border-y border-gray-100">
                <div className="flex items-start gap-3">
                   <div className="p-2 rounded-lg bg-gray-50">
                     <MapPin className="w-5 h-5 text-gray-600" />
                   </div>
                   <div className="flex-1 min-w-0">
                     <p className="text-sm text-gray-900 font-medium truncate">{facility.address}</p>
                     <p className="text-xs text-gray-500 mt-0.5">{facility.distanceKm.toFixed(1)} miles away</p>
                   </div>
                </div>
                
                <div className="flex items-start gap-3">
                   <div className="p-2 rounded-lg bg-gray-50">
                     <Calendar className="w-5 h-5 text-gray-600" />
                   </div>
                   <div className="flex-1 min-w-0">
                     <p className="text-sm text-gray-900 font-medium">{formatNextAvailable(facility.nextAvailableAt)}</p>
                     <p className="text-xs text-gray-500 mt-0.5">Next available</p>
                   </div>
                </div>

                <div className="flex items-start gap-3">
                   <div className="p-2 rounded-lg bg-gray-50">
                     <Clock className="w-5 h-5 text-gray-600" />
                   </div>
                   <div className="flex-1 min-w-0">
                     <p className="text-sm text-gray-900 font-medium">{facility.avgWaitMinutes || '20'} min</p>
                     <p className="text-xs text-gray-500 mt-0.5">Avg. wait time</p>
                   </div>
                </div>
              </div>

              {/* Services and Insurance */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 pt-4">
                 <div className="flex items-start gap-3">
                    <div className="mt-0.5">
                      <FileText className="w-5 h-5 text-gray-500" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <span className="text-sm font-medium text-gray-700">Services: </span>
                      <span className="text-sm text-gray-900">
                        {facility.services.slice(0, 3).join(", ")}
                        {facility.services.length > 3 && ", ..."}
                      </span>
                    </div>
                 </div>

                 <div className="flex items-start gap-3">
                    <div className="mt-0.5">
                      <ShieldCheck className="w-5 h-5 text-gray-500" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <span className="text-sm font-medium text-gray-700">Insurances: </span>
                      <span className="text-sm text-gray-900">
                        {facility.insurance.slice(0, 2).join(", ")}
                        {facility.insurance.length > 2 && ", ..."}
                      </span>
                    </div>
                 </div>
              </div>

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
