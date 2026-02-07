import { MapPin, DollarSign, Calendar, Clock, Star, CheckCircle2 } from "lucide-react";

export interface UIFacility {
  id: string;
  name: string;
  type: string;
  distance: number;
  price: number;
  rating: number;
  reviews: number;
  nextAvailable: string;
  address: string;
  insurance: string[];
  services: string[];
  urgent: boolean;
  capacity: string;
  waitTime: string;
  lat?: number;
  lon?: number;
}

interface SearchResultsProps {
  facilities: UIFacility[];
  loading: boolean;
  onSelectFacility: (facility: UIFacility) => void;
}

export function SearchResults({ facilities, loading, onSelectFacility }: SearchResultsProps) {
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

      <div className="space-y-4">
        {facilities.map((facility) => (
          <div
            key={facility.id}
            className="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-lg transition-shadow cursor-pointer"
            onClick={() => onSelectFacility(facility)}
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-start gap-3">
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="text-lg font-semibold text-gray-900">
                        {facility.name}
                      </h3>
                      {facility.urgent && (
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
                <div className="grid grid-cols-4 gap-4 mb-4">
                  <div className="flex items-start gap-2">
                    <MapPin className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {facility.distance.toFixed(1)} km away
                      </p>
                      <p className="text-xs text-gray-600">{facility.address}</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <DollarSign className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        ₦{facility.price.toLocaleString()}
                      </p>
                      <p className="text-xs text-gray-600">Estimated cost</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <Calendar className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {facility.nextAvailable}
                      </p>
                      <p className="text-xs text-gray-600">Next available</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <Clock className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {facility.waitTime}
                      </p>
                      <p className="text-xs text-gray-600">Avg. wait time</p>
                    </div>
                  </div>
                </div>

                {/* Services & Insurance */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="w-4 h-4 text-green-600" />
                    <span className="text-sm text-gray-700">
                      Services: {facility.services.join(", ")}
                    </span>
                  </div>
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="w-4 h-4 text-green-600" />
                    <span className="text-sm text-gray-700">
                      Insurance: {facility.insurance.slice(0, 3).join(", ")}
                      {facility.insurance.length > 3 && ` +${facility.insurance.length - 3} more`}
                    </span>
                  </div>
                </div>
              </div>

              {/* Right Side - Capacity Status */}
              <div className="text-right">
                <div
                  className={`inline-flex items-center gap-2 px-3 py-2 rounded-lg mb-6 ${
                    facility.capacity === "Available"
                      ? "bg-green-100 text-green-800"
                      : "bg-yellow-100 text-yellow-800"
                  }`}
                >
                  <div
                    className={`w-2 h-2 rounded-full ${
                      facility.capacity === "Available" ? "bg-green-600" : "bg-yellow-600"
                    }`}
                  />
                  <span className="text-sm font-medium">{facility.capacity}</span>
                </div>
                <button className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
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
