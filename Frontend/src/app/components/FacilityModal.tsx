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
  preSelectedServiceName?: string;
}

export function FacilityModal({ facility, onClose, preSelectedServiceName }: FacilityModalProps) {
  const [slots, setSlots] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedSlot, setSelectedSlot] = useState<any>(null);
  const [booking, setBooking] = useState(false);
  const [expandedServiceId, setExpandedServiceId] = useState<string | null>(
    preSelectedServiceName ? preSelectedServiceName : null
  );

  // Primary booker details
  const [patientName, setPatientName] = useState("");
  const [patientEmail, setPatientEmail] = useState("");
  const [patientPhone, setPatientPhone] = useState("");
  const [patientNIN, setPatientNIN] = useState("");
  const [whatsappOptIn, setWhatsappOptIn] = useState(true);
  
  // Booking on behalf toggle
  const [bookingOnBehalf, setBookingOnBehalf] = useState(false);
  const [beneficiaryName, setBeneficiaryName] = useState("");
  const [beneficiaryNIN, setBeneficiaryNIN] = useState("");
  
  // Special needs/accommodations
  const [specialNeeds, setSpecialNeeds] = useState("");
  
  // State for "All Services" modal
  const [showAllServices, setShowAllServices] = useState(false);
  const [serviceSearchQuery, setServiceSearchQuery] = useState("");

  const [serviceEnrichments, setServiceEnrichments] = useState<Record<string, ProcedureEnrichment>>({});
  const [serviceEnrichmentLoading, setServiceEnrichmentLoading] = useState<Record<string, boolean>>({});

  // SSE for real-time updates
  const { data: sseData, isConnected: sseConnected } = useSSEFacility(facility.id);

  const formatCurrency = (value: number, currency?: string | null) => {
    const symbol = currency === "NGN" ? "₦" : currency === "USD" ? "$" : currency ? `${currency} ` : "₦";
    return `${symbol}${value.toLocaleString("en-NG")}`;
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

  // Auto-scroll to pre-selected service
  useEffect(() => {
    if (preSelectedServiceName) {
      setTimeout(() => {
        const element = document.getElementById(`service-${preSelectedServiceName}`);
        if (element) {
          element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
        }
      }, 100);
    }
  }, [preSelectedServiceName]);

  // Update from SSE events
  useEffect(() => {
    // Could add SSE-based facility updates here if needed
  }, [sseData]);

  const handleBook = async () => {
    if (!selectedSlot) return;

    const emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(patientEmail.trim());
    const phoneValid = !whatsappOptIn || /^\+?[1-9]\d{7,14}$/.test(patientPhone.trim());
    const ninValid = patientNIN.trim().length === 11 && /^\d{11}$/.test(patientNIN.trim());
    
    // Validate booking on behalf fields
    if (bookingOnBehalf) {
      const beneficiaryNINValid = beneficiaryNIN.trim().length === 11 && /^\d{11}$/.test(beneficiaryNIN.trim());
      if (!beneficiaryName.trim() || !beneficiaryNINValid) {
        alert("Please provide valid beneficiary name and 11-digit NIN.");
        return;
      }
    }

    if (!patientName.trim() || !emailValid || !phoneValid || !ninValid) {
      alert("Please provide valid name, email, phone number, and 11-digit NIN.");
      return;
    }

    setBooking(true);
    try {
      await api.bookAppointment({
        facility_id: facility.id,
        scheduled_at: selectedSlot.start_time,
        patient_name: patientName.trim(),
        patient_email: patientEmail.trim(),
        patient_phone: whatsappOptIn ? patientPhone.trim() : "",
        patient_nin: patientNIN.trim(),
        booking_on_behalf: bookingOnBehalf,
        beneficiary_name: bookingOnBehalf ? beneficiaryName.trim() : undefined,
        beneficiary_nin: bookingOnBehalf ? beneficiaryNIN.trim() : undefined,
        special_needs: specialNeeds.trim() || undefined,
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

  const emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(patientEmail.trim());
  const phoneValid = !whatsappOptIn || /^\+?[1-9]\d{7,14}$/.test(patientPhone.trim());
  const ninValid = patientNIN.trim().length === 11 && /^\d{11}$/.test(patientNIN.trim());
  const beneficiaryNINValid = !bookingOnBehalf || (beneficiaryNIN.trim().length === 11 && /^\d{11}$/.test(beneficiaryNIN.trim()));
  const canBook = !!selectedSlot && !booking && !!patientName.trim() && emailValid && phoneValid && ninValid && beneficiaryNINValid && (!bookingOnBehalf || !!beneficiaryName.trim());

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

            {/* Ward Capacity Status */}
            {facility.wardStatuses && Object.keys(facility.wardStatuses).length > 0 && (
              <section>
                <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wide mb-4 flex items-center gap-2">
                  <Activity className="w-3.5 h-3.5" />
                  Real-time Ward Capacity
                </h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {Object.entries(facility.wardStatuses).map(([wardId, data]) => (
                    <div key={wardId} className="flex items-center justify-between p-3 rounded-xl bg-gray-50 border border-gray-100">
                      <div>
                        <p className="text-xs font-semibold text-gray-500 uppercase tracking-tight">{wardId.replace(/_/g, ' ')}</p>
                        <div className="flex items-center gap-2 mt-1">
                          <span className={`text-sm font-bold ${
                            data.status === 'full' ? 'text-red-600' : 
                            data.status === 'busy' ? 'text-orange-600' : 
                            'text-green-600'
                          }`}>
                            {data.status.toUpperCase()}
                          </span>
                          {data.trend === 'increasing' && (
                            <span className="flex items-center text-[10px] text-red-500 font-bold bg-red-50 px-1.5 py-0.5 rounded">
                              <ChevronUp className="w-3 h-3" />
                              Spiking
                            </span>
                          )}
                          {data.trend === 'decreasing' && (
                            <span className="flex items-center text-[10px] text-blue-500 font-bold bg-blue-50 px-1.5 py-0.5 rounded">
                              <ChevronDown className="w-3 h-3" />
                              Calming
                            </span>
                          )}
                        </div>
                      </div>
                      <div classname="text-right">
                        <p className="text-lg font-bold text-gray-900">{data.count}</p>
                        <p className="text-[10px] text-gray-400">Transactions / 4h</p>
                        {data.estimatedWaitMinutes && (
                          <p className="text-[10px] font-medium text-blue-600 mt-1">
                            ~{data.estimatedWaitMinutes} min wait
                          </p>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            )}

             <hr className="border-gray-100" />
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
                         <div key={serviceKey} id={`service-${service.name}`} className="bg-white scroll-mt-4">
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

            <hr className="border-gray-100" />

            {/* Booking Details */}
            <section>
              <h3 className="text-xs font-bold text-gray-500 uppercase tracking-wide mb-4">Booking details</h3>
              
              {/* Primary Booker Information */}
              <div className="space-y-4 mb-6 pb-6 border-b border-gray-100">
                <h4 className="text-sm font-semibold text-gray-900">Your Information</h4>
                
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="md:col-span-2">
                    <label className="text-xs text-gray-500 mb-1 block">Full name *</label>
                    <input
                      type="text"
                      value={patientName}
                      onChange={(e) => setPatientName(e.target.value)}
                      placeholder="e.g., Ada Okafor"
                      className="w-full px-3 py-2.5 rounded-lg border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                  
                  <div>
                    <label className="text-xs text-gray-500 mb-1 block">Email *</label>
                    <input
                      type="email"
                      value={patientEmail}
                      onChange={(e) => setPatientEmail(e.target.value)}
                      placeholder="e.g., ada@example.com"
                      className={`w-full px-3 py-2.5 rounded-lg border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                        patientEmail && !emailValid ? "border-red-300" : "border-gray-200"
                      }`}
                    />
                    {patientEmail && !emailValid && (
                      <p className="mt-1 text-xs text-red-500">Enter a valid email address.</p>
                    )}
                  </div>
                  
                  <div>
                    <label className="text-xs text-gray-500 mb-1 block">Phone (with country code) *</label>
                    <input
                      type="tel"
                      value={patientPhone}
                      onChange={(e) => setPatientPhone(e.target.value)}
                      placeholder="e.g., +2348012345678"
                      className={`w-full px-3 py-2.5 rounded-lg border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                        patientPhone && !phoneValid ? "border-red-300" : "border-gray-200"
                      }`}
                    />
                    {patientPhone && !phoneValid && (
                      <p className="mt-1 text-xs text-red-500">Use E.164 format (e.g., +2348012345678).</p>
                    )}
                  </div>
                  
                  <div className="md:col-span-2">
                    <label className="text-xs text-gray-500 mb-1 block">National Identification Number (NIN) - 11 digits *</label>
                    <input
                      type="text"
                      value={patientNIN}
                      onChange={(e) => setPatientNIN(e.target.value.replace(/\D/g, '').slice(0, 11))}
                      placeholder="e.g., 12345678901"
                      maxLength={11}
                      className={`w-full px-3 py-2.5 rounded-lg border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                        patientNIN && !ninValid ? "border-red-300" : "border-gray-200"
                      }`}
                    />
                    {patientNIN && !ninValid && (
                      <p className="mt-1 text-xs text-red-500">NIN must be exactly 11 digits.</p>
                    )}
                    <p className="mt-1 text-xs text-gray-400">Your unique National Identification Number from NIMC</p>
                  </div>
                  
                  <div className="md:col-span-2 flex items-start gap-2">
                    <input
                      id="whatsapp-opt-in"
                      type="checkbox"
                      checked={whatsappOptIn}
                      onChange={(e) => setWhatsappOptIn(e.target.checked)}
                      className="mt-1 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <label htmlFor="whatsapp-opt-in" className="text-xs text-gray-600">
                      I agree to receive appointment updates via WhatsApp.
                    </label>
                  </div>
                </div>
              </div>
              
              {/* Booking on Behalf Section */}
              <div className="space-y-4 mb-6 pb-6 border-b border-gray-100">
                <div className="flex items-start gap-2">
                  <input
                    id="booking-behalf"
                    type="checkbox"
                    checked={bookingOnBehalf}
                    onChange={(e) => setBookingOnBehalf(e.target.checked)}
                    className="mt-1 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                  <label htmlFor="booking-behalf" className="text-sm font-semibold text-gray-900">
                    I'm booking this appointment on behalf of someone else
                  </label>
                </div>
                
                {bookingOnBehalf && (
                  <div className="ml-6 pt-2 space-y-4 p-4 bg-blue-50 rounded-lg border border-blue-100">
                    <h4 className="text-sm font-semibold text-gray-900">Beneficiary Information</h4>
                    
                    <div>
                      <label className="text-xs text-gray-500 mb-1 block">Beneficiary's full name *</label>
                      <input
                        type="text"
                        value={beneficiaryName}
                        onChange={(e) => setBeneficiaryName(e.target.value)}
                        placeholder="e.g., Chidi Nwankwo"
                        className="w-full px-3 py-2.5 rounded-lg border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      />
                    </div>
                    
                    <div>
                      <label className="text-xs text-gray-500 mb-1 block">Beneficiary's NIN - 11 digits *</label>
                      <input
                        type="text"
                        value={beneficiaryNIN}
                        onChange={(e) => setBeneficiaryNIN(e.target.value.replace(/\D/g, '').slice(0, 11))}
                        placeholder="e.g., 98765432109"
                        maxLength={11}
                        className={`w-full px-3 py-2.5 rounded-lg border text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 ${
                          beneficiaryNIN && !beneficiaryNINValid ? "border-red-300" : "border-gray-200"
                        }`}
                      />
                      {beneficiaryNIN && !beneficiaryNINValid && (
                        <p className="mt-1 text-xs text-red-500">NIN must be exactly 11 digits.</p>
                      )}
                    </div>
                  </div>
                )}
              </div>
              
              {/* Special Needs/Accommodations */}
              <div className="space-y-4">
                <div>
                  <label className="text-xs text-gray-500 mb-1 block">Special needs or accommodation requests</label>
                  <textarea
                    value={specialNeeds}
                    onChange={(e) => setSpecialNeeds(e.target.value)}
                    placeholder="e.g., Wheelchair accessibility, Sign language interpreter, Physical therapy, etc. (Optional)"
                    rows={3}
                    className="w-full px-3 py-2.5 rounded-lg border border-gray-200 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
                  />
                  <p className="mt-1 text-xs text-gray-400">Let us know about any special accommodations you may need during your visit.</p>
                </div>
              </div>
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
                  disabled={!canBook}
                    className={`px-6 py-2.5 rounded-lg text-sm font-semibold text-white transition-colors shadow-sm ${
                    !canBook 
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