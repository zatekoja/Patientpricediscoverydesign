import { useState, useEffect } from "react";
import {
  X,
  MapPin,
  Calendar,
  Clock,
  Star,
  Phone,
  Globe,
  ChevronDown,
  ChevronUp,
  Search,
  CheckCircle2,
  Activity,
  AlertCircle,
} from "lucide-react";
import { api } from "../../lib/api";
import type { ProcedureEnrichment, ProviderHealthResponse } from "../../types/api";
import { useSSEFacility } from "../../lib/hooks/useSSE";

interface FacilityModalProps {
  facility: {
    id: string;
    name: string;
    type: string;
    distanceKm: number;
    priceMin?: number | null;
    priceMax?: number | null;
    currency?: string | null;
    rating: number;
    reviews: number;
    address: string;
    phoneNumber?: string | null;
    website?: string | null;
    services: string[];
    servicePrices: {
      procedureId?: string;
      name: string;
      price: number;
      currency: string;
      description?: string;
      category?: string;
      code?: string;
      estimatedDuration?: number;
    }[];
    insurance: string[];
    capacityStatus?: string | null;
    avgWaitMinutes?: number | null;
    urgentCareAvailable?: boolean | null;
    nextAvailableAt?: string | null;
  };
  onClose: () => void;
}

export function FacilityModal({ facility, onClose }: FacilityModalProps) {
  const [slots, setSlots] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedSlot, setSelectedSlot] = useState<any>(null);
  const [booking, setBooking] = useState(false);
  const [expandedServiceId, setExpandedServiceId] = useState<string | null>(null);
  
  // State for "All Services" modal
  const [showAllServices, setShowAllServices] = useState(false);
  const [serviceSearchQuery, setServiceSearchQuery] = useState("");

  const [serviceEnrichments, setServiceEnrichments] = useState<Record<string, ProcedureEnrichment>>({});
  const [serviceEnrichmentLoading, setServiceEnrichmentLoading] = useState<Record<string, boolean>>({});

  // Provider health state
  const [providerHealth, setProviderHealth] = useState<ProviderHealthResponse | null>(null);
  const [providerHealthLoading, setProviderHealthLoading] = useState(false);

  // SSE for real-time updates
  const { data: sseData, isConnected: sseConnected } = useSSEFacility(facility.id);

  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : currency ? `${currency} ` : "₦";
    return `${symbol}${Math.round(value).toLocaleString()}`;
  };

  const loadEnrichment = async (procedureId: string) => {
    if (!procedureId || serviceEnrichmentLoading[procedureId] || serviceEnrichments[procedureId]) {
      return;
    }

    setServiceEnrichmentLoading((prev) => ({ ...prev, [procedureId]: true }));
    try {
      const enrichment = await api.getProcedureEnrichment(procedureId);
      setServiceEnrichments((prev) => ({ ...prev, [procedureId]: enrichment }));
    } catch (err) {
      console.error("Failed to load procedure enrichment:", err);
    } finally {
      setServiceEnrichmentLoading((prev) => ({ ...prev, [procedureId]: false }));
    }
  };

  useEffect(() => {
    const fetchSlots = async () => {
      setLoading(true);
      try {
        const from = new Date();
        const to = new Date();
        to.setDate(to.getDate() + 7);
        const res = await api.getAvailability(facility.id, from, to);
        setSlots(res.slots || []);
      } catch (err) {
        console.error("Failed to fetch slots:", err);
      } finally {
        setLoading(false);
      }
    };
    fetchSlots();
  }, [facility.id]);

  // Fetch provider health
  useEffect(() => {
    const fetchProviderHealth = async () => {
      setProviderHealthLoading(true);
      try {
        // Fetch health for megalek provider (hardcoded ID or make it dynamic)
        const health = await api.getProviderHealth("megalek");
        setProviderHealth(health);
      } catch (err) {
        console.error("Failed to fetch provider health:", err);
      } finally {
        setProviderHealthLoading(false);
      }
    };
    fetchProviderHealth();
  }, []);

  // Update from SSE events
  useEffect(() => {
    if (sseData && sseData.event_type === 'service_health_update') {
      // Update provider health from SSE
      setProviderHealth(prev => ({
        ...prev,
        healthy: sseData.changed_fields?.healthy ?? prev?.healthy ?? false,
        lastSync: sseData.timestamp,
        message: sseData.changed_fields?.message ?? prev?.message,
      }));
    }
  }, [sseData]);

  const handleBook = async () => {
    if (!selectedSlot) return;
    setBooking(true);
    try {
      await api.bookAppointment({
        facility_id: facility.id,
        scheduled_at: selectedSlot.start_time,
        patient_name: "Demo User", // Should come from auth/form
        patient_email: "demo@example.com",
      });
      alert("Appointment booked successfully!");
      onClose();
    } catch (err) {
      console.error("Booking failed:", err);
      alert("Failed to book appointment.");
    } finally {
      setBooking(false);
    }
  };

  // Group slots by date
  const groupedSlots = slots.reduce((acc: any, slot: any) => {
    const dateObj = new Date(slot.start_time);
    const date = dateObj.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric' });
    if (!acc[date]) acc[date] = [];
    acc[date].push(slot);
    return acc;
  }, {});

  const priceRangeDisplay = facility.priceMin 
  ? `${formatCurrency(facility.priceMin, facility.currency)} - ${facility.priceMax ? formatCurrency(facility.priceMax, facility.currency) : '...'}`
  : "Price Varies";

  const nextAvailableDisplay = facility.nextAvailableAt 
    ? new Date(facility.nextAvailableAt).toLocaleString('en-NG', { weekday: 'long', hour: '2-digit', minute: '2-digit' })
    : "Next available";

  // Filter services for the "All Services" modal
  const filteredServices = facility.servicePrices.filter(s => 
    s.name.toLowerCase().includes(serviceSearchQuery.toLowerCase())
  );

  return (
    <div 
      className="fixed inset-0 bg-black/30 backdrop-blur-[2px] flex items-center justify-center z-50 p-4"
      onClick={onClose}
    >
      <div 
        className="bg-white rounded-2xl shadow-2xl w-full max-w-3xl max-h-[90vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        
        {/* Header Section */}
        <div className="p-6 border-b border-gray-100 pb-4">
            <div className="flex justify-between items-start mb-2">
                <div>
                   <h2 className="text-2xl font-bold text-gray-900">{facility.name}</h2>
                   <div className="flex items-center gap-2 mt-1">
                      <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide">{facility.type}</span>
                      <div className="flex items-center gap-1">
                        <Star className="w-3.5 h-3.5 text-yellow-400 fill-yellow-400" />
                        <span className="text-sm font-medium text-gray-900">{facility.rating}</span>
                      </div>
                      {facility.urgentCareAvailable && (
                        <span className="px-2 py-0.5 bg-blue-50 text-blue-600 text-[10px] font-semibold rounded uppercase tracking-wide border border-blue-100">
                          Urgent care available
                        </span>
                      )}
                       {facility.capacityStatus?.toLowerCase().includes("available") && (
                        <span className="flex items-center gap-1 px-2 py-0.5 bg-green-50 text-green-700 text-[10px] font-semibold rounded uppercase tracking-wide border border-green-100">
                          <span className="w-1.5 h-1.5 rounded-full bg-green-500"></span>
                          Available
                        </span>
                      )}
                   </div>
                </div>
                <div className="text-right">
                    <p className="text-xl font-bold text-gray-900">{priceRangeDisplay}</p>
                     <button
                        onClick={onClose}
                        className="text-gray-400 hover:text-gray-600 p-1 absolute top-4 right-4"
                    >
                        <X className="w-6 h-6" />
                    </button>
                </div>
            </div>
        </div>

        {/* Scrollable Content */}
        <div className="flex-1 overflow-y-auto p-6 space-y-8">
            
            {/* Company Information */}
            <section>
                <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wide mb-4">Company Information</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-y-4 gap-x-8">
                    {/* Address */}
                    <div className="flex items-start gap-3">
                        <MapPin className="w-5 h-5 text-gray-400 mt-0.5" />
                        <div>
                            <p className="text-xs text-gray-500 mb-0.5">Address</p>
                            <p className="text-sm font-medium text-gray-900 underline decoration-gray-300 underline-offset-2">{facility.address}</p>
                        </div>
                    </div>
                     {/* Next Available */}
                    <div className="flex items-start gap-3">
                        <Calendar className="w-5 h-5 text-gray-400 mt-0.5" />
                         <div>
                            <p className="text-xs text-gray-500 mb-0.5">Next available</p>
                            <p className="text-sm font-medium text-gray-900">{nextAvailableDisplay}</p>
                        </div>
                    </div>
                     {/* Phone */}
                    <div className="flex items-start gap-3">
                        <Phone className="w-5 h-5 text-gray-400 mt-0.5" />
                        <div>
                            <p className="text-xs text-gray-500 mb-0.5">Phone number</p>
                            <p className="text-sm font-medium text-gray-900 underline decoration-gray-300 underline-offset-2">{facility.phoneNumber || "N/A"}</p>
                        </div>
                    </div>
                     {/* Wait Time */}
                    <div className="flex items-start gap-3">
                        <Clock className="w-5 h-5 text-gray-400 mt-0.5" />
                        <div>
                            <p className="text-xs text-gray-500 mb-0.5">Avg. wait time</p>
                            <p className="text-sm font-medium text-gray-900">{facility.avgWaitMinutes ? `${facility.avgWaitMinutes} min` : "N/A"}</p>
                        </div>
                    </div>
                     {/* Website */}
                    <div className="flex items-start gap-3 md:col-span-2">
                         <Globe className="w-5 h-5 text-gray-400 mt-0.5" />
                         <div>
                            <p className="text-xs text-gray-500 mb-0.5">Website</p>
                            <a href={facility.website || "#"} target="_blank" rel="noreferrer" className="text-sm font-medium text-gray-900 underline decoration-gray-300 underline-offset-2">{facility.website || "N/A"}</a>
                        </div>
                    </div>
                </div>
            </section>

             <hr className="border-gray-100" />

            {/* Available Services */}
             <section>
                 <div className="flex items-center justify-between mb-4">
                    <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wide">Available Services</h3>
                    <button 
                      onClick={() => setShowAllServices(true)}
                      className="text-sm text-blue-600 hover:text-blue-800 font-medium"
                    >
                      See all
                    </button>
                 </div>
                 
                 <div className="border border-gray-200 rounded-lg divide-y divide-gray-200">
                    {facility.servicePrices.slice(0, 5).map((service, index) => { // Limit to 5 for prototype
                       const serviceKey = service.procedureId || service.name || String(index);
                       const isExpanded = expandedServiceId === serviceKey;
                       const enrichment = service.procedureId ? serviceEnrichments[service.procedureId] : undefined;
                       
                       return (
                         <div key={serviceKey} className="bg-white">
                            <button 
                                onClick={() => {
                                    const nextId = isExpanded ? null : serviceKey;
                                    setExpandedServiceId(nextId);
                                    if (!isExpanded && service.procedureId) {
                                      loadEnrichment(service.procedureId);
                                    }
                                }}
                                className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors text-left"
                            >
                                <span className="text-sm font-medium text-gray-900">{service.name}</span>
                                <div className="flex items-center gap-4">
                                    <span className="text-sm font-medium text-gray-900">{formatCurrency(service.price, service.currency)}</span>
                                    {isExpanded ? <ChevronUp className="w-4 h-4 text-gray-400" /> : <ChevronDown className="w-4 h-4 text-gray-400" />}
                                </div>
                            </button>
                            {isExpanded && (
                                <div className="px-4 pb-4 text-sm text-gray-600 bg-gray-50/50">
                                   <div className="pt-2 border-t border-gray-100">
                                      <p className="mb-3 text-gray-800">{enrichment?.description || service.description || "No description available."}</p>
                                      <div className="grid grid-cols-2 gap-4 text-xs">
                                          <div>
                                              <span className="text-gray-500">Category:</span> <span className="font-medium text-gray-900">{service.category || "General"}</span>
                                          </div>
                                          <div>
                                               <span className="text-gray-500">Code:</span> <span className="font-medium text-gray-900">{service.code || "N/A"}</span>
                                          </div>
                                           <div>
                                               <span className="text-gray-500">Estimated time:</span> <span className="font-medium text-gray-900">{service.estimatedDuration ? `${service.estimatedDuration} mins` : "Varies"}</span>
                                          </div>
                                      </div>
                                   </div>
                                </div>
                            )}
                         </div>
                       );
                    })}
                     {facility.servicePrices.length === 0 && (
                         <div className="p-4 text-sm text-gray-500 text-center">No specific services listed.</div>
                     )}
                 </div>
             </section>

             <hr className="border-gray-100" />

            {/* Provider Health Status */}
            <section>
                <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wide mb-4 flex items-center gap-2">
                  <Activity className="w-4 h-4" />
                  Provider Health Status
                </h3>
                {providerHealthLoading ? (
                  <div className="text-sm text-gray-500">Loading provider status...</div>
                ) : providerHealth ? (
                  <div className="space-y-3">
                    <div className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                      <div className="flex items-center gap-3">
                        {providerHealth.healthy ? (
                          <CheckCircle2 className="w-5 h-5 text-green-600" />
                        ) : (
                          <AlertCircle className="w-5 h-5 text-red-600" />
                        )}
                        <div>
                          <p className="text-sm font-medium text-gray-900">
                            {providerHealth.healthy ? "Megalek Provider Active" : "Megalek Provider Issue"}
                          </p>
                          {providerHealth.message && (
                            <p className="text-xs text-gray-500 mt-0.5">{providerHealth.message}</p>
                          )}
                        </div>
                      </div>
                      {sseConnected && (
                        <div className="flex items-center gap-1.5 text-xs text-green-600 bg-green-50 px-2 py-1 rounded-full">
                          <div className="w-2 h-2 bg-green-600 rounded-full animate-pulse"></div>
                          Live
                        </div>
                      )}
                    </div>
                    {providerHealth.lastSync && (
                      <p className="text-xs text-gray-500">
                        Last synced: {new Date(providerHealth.lastSync).toLocaleString()}
                      </p>
                    )}
                    <div className="grid grid-cols-2 gap-3 text-sm">
                      <div className="p-3 bg-blue-50 rounded-lg">
                        <p className="text-xs text-gray-600 mb-1">Capacity Status</p>
                        <p className="font-semibold text-blue-900">
                          {facility.capacityStatus || "Normal"}
                        </p>
                      </div>
                      <div className="p-3 bg-purple-50 rounded-lg">
                        <p className="text-xs text-gray-600 mb-1">Avg Wait Time</p>
                        <p className="font-semibold text-purple-900">
                          {facility.avgWaitMinutes ? `${facility.avgWaitMinutes} mins` : "N/A"}
                        </p>
                      </div>
                    </div>
                    <p className="text-xs text-gray-500 italic">
                      Provider health powers real-time capacity and wait time updates via SSE
                    </p>
                  </div>
                ) : (
                  <p className="text-sm text-gray-500">Provider health unavailable</p>
                )}
            </section>

             <hr className="border-gray-100" />

            {/* Available Time Slots */}
            <section>
                 <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wide mb-4">Available Time Slots</h3>
                 {loading ? (
                     <div className="text-sm text-gray-500">Loading slots...</div>
                 ) : Object.keys(groupedSlots).length > 0 ? (
                     <div className="space-y-6">
                         {Object.entries(groupedSlots).slice(0, 2).map(([date, dateSlots]: [string, any]) => (
                             <div key={date}>
                                 <p className="text-sm font-medium text-gray-700 mb-3">{date}</p>
                                 <div className="flex flex-wrap gap-2">
                                     {dateSlots.map((slot: any, idx: number) => (
                                         <button
                                            key={idx}
                                            onClick={() => setSelectedSlot(slot)}
                                            className={`px-4 py-2 rounded-lg border text-xs font-medium transition-all ${
                                                selectedSlot === slot 
                                                ? "bg-blue-600 text-white border-blue-600 shadow-sm"
                                                : "bg-white text-gray-700 border-gray-200 hover:border-blue-300 hover:text-blue-600"
                                            }`}
                                         >
                                             {new Date(slot.start_time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                         </button>
                                     ))}
                                 </div>
                             </div>
                         ))}
                     </div>
                 ) : (
                     <p className="text-sm text-gray-500">No time slots available currently.</p>
                 )}
            </section>
        </div>

        {/* Footer Actions */}
        <div className="p-6 border-t border-gray-100 bg-white">
            <div className="flex gap-4 justify-end">
                <button className="px-6 py-2.5 rounded-lg border border-gray-300 text-sm font-semibold text-gray-700 hover:bg-gray-50 transition-colors">
                    Call business
                </button>
                <button 
                    onClick={handleBook}
                    disabled={!selectedSlot || booking}
                    className={`px-6 py-2.5 rounded-lg text-sm font-semibold text-white transition-colors shadow-sm ${
                        !selectedSlot || booking 
                        ? "bg-gray-300 cursor-not-allowed" 
                        : "bg-blue-600 hover:bg-blue-700"
                    }`}
                >
                    {booking ? "Booking..." : "Book appointment"}
                </button>
            </div>
        </div>

      </div>

      {/* All Services Modal - Overlay */}
      {showAllServices && (
        <div 
          className="fixed inset-0 z-[60] flex items-center justify-center bg-black/40 backdrop-blur-sm p-4"
          onClick={() => setShowAllServices(false)}
        >
          <div 
            className="bg-white rounded-2xl shadow-xl w-full max-w-[500px] h-[600px] flex flex-col overflow-hidden animate-in zoom-in-95 duration-200"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div className="flex items-center justify-between p-6 pb-2">
              <h2 className="text-xl font-semibold text-gray-900">All services</h2>
              <button
                onClick={() => setShowAllServices(false)}
                className="text-gray-400 hover:text-gray-600 transition-colors"
              >
                <X className="w-6 h-6" />
              </button>
            </div>

            {/* Search Bar */}
            <div className="px-6 py-4">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 w-5 h-5" />
                <input
                  type="text"
                  placeholder="Search for services"
                  value={serviceSearchQuery}
                  onChange={(e) => setServiceSearchQuery(e.target.value)}
                  className="w-full pl-10 pr-4 py-3 bg-white border border-gray-200 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all shadow-sm"
                />
              </div>
            </div>

            {/* Divider */}
            <div className="h-px bg-gray-100 mx-6 mb-2"></div>

            {/* Services List */}
            <div className="flex-1 overflow-y-auto px-6 pb-6 space-y-3">
              {filteredServices.length > 0 ? (
                filteredServices.map((service, index) => {
                  const serviceKey = service.procedureId || service.name || String(index);
                  const isExpanded = expandedServiceId === serviceKey;
                  const enrichment = service.procedureId ? serviceEnrichments[service.procedureId] : undefined;
                  
                  return (
                    <div 
                      key={serviceKey} 
                      className="border border-gray-200 rounded-xl overflow-hidden hover:border-blue-200 transition-colors"
                    >
                      <button
                        onClick={() => {
                           const nextId = isExpanded ? null : serviceKey;
                           setExpandedServiceId(nextId);
                           if (!isExpanded && service.procedureId) {
                              loadEnrichment(service.procedureId);
                           }
                        }}
                        className="w-full flex items-center justify-between p-4 bg-white text-left"
                      >
                        <div className="flex items-center gap-3">
                          <CheckCircle2 className="w-5 h-5 text-green-500 flex-shrink-0" />
                          <span className="text-sm font-medium text-gray-900">{service.name}</span>
                        </div>
                        <div className="flex items-center gap-3">
                           <span className="text-sm font-semibold text-gray-900">{formatCurrency(service.price, service.currency)}</span>
                           {isExpanded ? <ChevronUp className="w-4 h-4 text-gray-400" /> : <ChevronDown className="w-4 h-4 text-gray-400" />}
                        </div>
                      </button>
                      
                      {/* Expanded Details in Modal */}
                      {isExpanded && (
                         <div className="bg-gray-50 border-t border-gray-100 p-4 text-sm text-gray-600">
                             <p className="mb-2">{enrichment?.description || service.description || "No description available."}</p>
                             <div className="flex flex-wrap gap-x-4 gap-y-2 text-xs text-gray-500">
                                {service.category && <span>Category: <strong className="text-gray-700">{service.category}</strong></span>}
                                {service.code && <span>Code: <strong className="text-gray-700">{service.code}</strong></span>}
                                {service.estimatedDuration && <span>Duration: <strong className="text-gray-700">{service.estimatedDuration} min</strong></span>}
                             </div>
                         </div>
                      )}
                    </div>
                  );
                })
              ) : (
                <div className="text-center py-12 text-gray-500">
                  <p>No services found matching "{serviceSearchQuery}"</p>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}