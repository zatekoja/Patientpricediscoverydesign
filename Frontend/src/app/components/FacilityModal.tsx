import { useState, useEffect } from "react";
import {
  X,
  MapPin,
  DollarSign,
  Calendar,
  Clock,
  Star,
  Phone,
  Globe,
  CheckCircle2,
  AlertCircle,
  ChevronRight,
} from "lucide-react";
import { api } from "../../lib/api";

interface FacilityModalProps {
  facility: any;
  onClose: () => void;
}

export function FacilityModal({ facility, onClose }: FacilityModalProps) {
  const [slots, setSlots] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedSlot, setSelectedSlot] = useState<any>(null);
  const [booking, setBooking] = useState(false);

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
    const date = new Date(slot.start_time).toLocaleDateString('en-US', { weekday: 'long', month: 'short', day: 'numeric' });
    if (!acc[date]) acc[date] = [];
    acc[date].push(slot);
    return acc;
  }, {});

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg max-w-4xl w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <h2 className="text-2xl font-bold text-gray-900">{facility.name}</h2>
              {facility.urgent && (
                <span className="px-2 py-1 bg-red-100 text-red-700 text-xs rounded-full">
                  Urgent Care Available
                </span>
              )}
            </div>
            <p className="text-gray-600">{facility.type}</p>
            <div className="flex items-center gap-1 mt-2">
              <Star className="w-4 h-4 text-yellow-400 fill-yellow-400" />
              <span className="font-semibold text-gray-900">{facility.rating}</span>
              <span className="text-sm text-gray-600">
                ({facility.reviews} reviews)
              </span>
            </div>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600 p-1"
          >
            <X className="w-6 h-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {/* Quick Info Cards */}
          <div className="grid grid-cols-4 gap-4 mb-6">
            <div className="bg-blue-50 rounded-lg p-4">
              <MapPin className="w-5 h-5 text-blue-600 mb-2" />
              <p className="text-2xl font-bold text-gray-900 mb-1">
                {facility.distance} km
              </p>
              <p className="text-sm text-gray-600">Distance</p>
            </div>
            <div className="bg-green-50 rounded-lg p-4">
              <DollarSign className="w-5 h-5 text-green-600 mb-2" />
              <p className="text-2xl font-bold text-gray-900 mb-1">
                ₦{facility.price.toLocaleString()}
              </p>
              <p className="text-sm text-gray-600">Estimated Cost</p>
            </div>
            <div className="bg-purple-50 rounded-lg p-4">
              <Clock className="w-5 h-5 text-purple-600 mb-2" />
              <p className="text-2xl font-bold text-gray-900 mb-1">
                {facility.waitTime}
              </p>
              <p className="text-sm text-gray-600">Avg. Wait</p>
            </div>
            <div
              className={`rounded-lg p-4 ${
                facility.capacity === "Available"
                  ? "bg-green-50"
                  : "bg-yellow-50"
              }`}
            >
              <AlertCircle
                className={`w-5 h-5 mb-2 ${
                  facility.capacity === "Available"
                    ? "text-green-600"
                    : "text-yellow-600"
                }`}
              />
              <p className="text-2xl font-bold text-gray-900 mb-1">
                {facility.capacity}
              </p>
              <p className="text-sm text-gray-600">Capacity</p>
            </div>
          </div>

          {/* Contact & Address */}
          <div className="bg-gray-50 rounded-lg p-4 mb-6">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="flex items-start gap-2 mb-3">
                  <MapPin className="w-5 h-5 text-gray-400 mt-0.5" />
                  <div>
                    <p className="font-semibold text-gray-900 mb-1">Address</p>
                    <p className="text-sm text-gray-600">{facility.address}</p>
                    <button className="text-sm text-blue-600 hover:underline mt-1">
                      Get Directions
                    </button>
                  </div>
                </div>
              </div>
              <div>
                <div className="flex items-start gap-2 mb-3">
                  <Phone className="w-5 h-5 text-gray-400 mt-0.5" />
                  <div>
                    <p className="font-semibold text-gray-900 mb-1">Phone</p>
                    <p className="text-sm text-gray-600">(555) 123-4567</p>
                    <button className="text-sm text-blue-600 hover:underline mt-1">
                      Call Now
                    </button>
                  </div>
                </div>
              </div>
            </div>
            <div className="flex items-start gap-2 pt-3 border-t border-gray-200">
              <Globe className="w-5 h-5 text-gray-400 mt-0.5" />
              <div>
                <p className="font-semibold text-gray-900 mb-1">Website</p>
                <a
                  href="#"
                  className="text-sm text-blue-600 hover:underline"
                >
                  www.{facility.name.toLowerCase().replace(/[^a-z]/g, "")}.com
                </a>
              </div>
            </div>
          </div>

          {/* Services */}
          <div className="mb-6">
            <h3 className="font-semibold text-gray-900 mb-3">
              Available Services
            </h3>
            <div className="grid grid-cols-2 gap-2">
              {facility.services.map((service: string, index: number) => (
                <div
                  key={index}
                  className="flex items-center gap-2 bg-white border border-gray-200 rounded-lg p-3"
                >
                  <CheckCircle2 className="w-5 h-5 text-green-600" />
                  <span className="text-sm text-gray-900">{service}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Insurance */}
          <div className="mb-6">
            <h3 className="font-semibold text-gray-900 mb-3">
              Accepted Insurance
            </h3>
            <div className="flex flex-wrap gap-2">
              {facility.insurance.map((ins: string, index: number) => (
                <span
                  key={index}
                  className="px-3 py-1 bg-blue-50 text-blue-700 rounded-full text-sm"
                >
                  {ins}
                </span>
              ))}
            </div>
          </div>

          {/* Available Appointments */}
          <div className="mb-6">
            <h3 className="font-semibold text-gray-900 mb-3">
              Available Appointment Times
            </h3>
            {loading ? (
              <p className="text-sm text-gray-500">Loading availability...</p>
            ) : Object.keys(groupedSlots).length > 0 ? (
              <div className="space-y-4">
                {Object.entries(groupedSlots).map(([date, dateSlots]: [string, any], index) => (
                  <div key={index}>
                    <p className="text-sm font-medium text-gray-700 mb-2">
                      {date}
                    </p>
                    <div className="grid grid-cols-4 gap-2">
                      {dateSlots.map((slot: any, slotIndex: number) => (
                        <button
                          key={slotIndex}
                          onClick={() => setSelectedSlot(slot)}
                          className={`px-4 py-2 border rounded-lg text-sm transition-colors ${
                            selectedSlot === slot
                              ? "bg-blue-600 text-white border-blue-600"
                              : "border-gray-300 hover:border-blue-500 hover:bg-blue-50"
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
              <p className="text-sm text-gray-500">No availability found for this facility.</p>
            )}
          </div>

          {/* Price Breakdown */}
          <div className="mb-6">
            <h3 className="font-semibold text-gray-900 mb-3">
              Price Breakdown (Estimated)
            </h3>
            <div className="bg-gray-50 rounded-lg p-4 space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Procedure Fee</span>
                <span className="text-gray-900">₦{(facility.price - 5000).toLocaleString()}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-gray-600">Consultation Fee</span>
                <span className="text-gray-900">₦5,000</span>
              </div>
              <div className="border-t border-gray-200 pt-2 flex justify-between font-semibold">
                <span className="text-gray-900">Total Estimated Cost</span>
                <span className="text-gray-900">₦{facility.price.toLocaleString()}</span>
              </div>
              <p className="text-xs text-gray-500 mt-2">
                * Final cost may vary based on insurance coverage and specific
                requirements
              </p>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3">
            <button
              onClick={handleBook}
              disabled={!selectedSlot || booking}
              className={`flex-1 px-6 py-3 rounded-lg transition-colors font-semibold flex items-center justify-center gap-2 ${
                !selectedSlot || booking
                  ? "bg-gray-300 cursor-not-allowed"
                  : "bg-blue-600 text-white hover:bg-blue-700"
              }`}
            >
              {booking ? "Booking..." : "Confirm Booking"}
              <ChevronRight className="w-5 h-5" />
            </button>
            <button className="px-6 py-3 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors font-semibold">
              Call Facility
            </button>
            <button className="px-6 py-3 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors font-semibold">
              Get Directions
            </button>
          </div>

          {/* Important Notice */}
          <div className="mt-6 bg-yellow-50 border border-yellow-200 rounded-lg p-4 flex gap-3">
            <AlertCircle className="w-5 h-5 text-yellow-600 flex-shrink-0 mt-0.5" />
            <div>
              <p className="font-semibold text-yellow-900 mb-1">
                Important Notice
              </p>
              <p className="text-sm text-yellow-800">
                Please call ahead to confirm availability and bring your insurance
                information and ID. Arrive 15 minutes before your appointment.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
