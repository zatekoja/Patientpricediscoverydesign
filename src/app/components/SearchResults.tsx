import { MapPin, DollarSign, Calendar, Clock, Star, CheckCircle2 } from "lucide-react";

interface SearchResultsProps {
  onSelectFacility: (facility: any) => void;
}

// Mock data for hospitals/facilities
const facilities = [
  {
    id: 1,
    name: "St. Mary's Medical Center",
    type: "Hospital",
    distance: 2.3,
    price: 450,
    rating: 4.8,
    reviews: 1243,
    nextAvailable: "Today, 3:00 PM",
    address: "123 Medical Drive, Downtown",
    insurance: ["Aetna", "Blue Cross", "Cigna", "Medicare"],
    services: ["MRI", "CT Scan", "X-Ray", "Ultrasound"],
    urgent: true,
    capacity: "Available",
    waitTime: "15 min"
  },
  {
    id: 2,
    name: "Central Imaging & Diagnostics",
    type: "Imaging Center",
    distance: 3.7,
    price: 350,
    rating: 4.9,
    reviews: 892,
    nextAvailable: "Tomorrow, 9:00 AM",
    address: "456 Radiology Blvd, Medical District",
    insurance: ["Aetna", "UnitedHealthcare", "Cigna"],
    services: ["MRI", "CT Scan", "PET Scan"],
    urgent: false,
    capacity: "Limited",
    waitTime: "30 min"
  },
  {
    id: 3,
    name: "Regional Emergency Hospital",
    type: "Hospital",
    distance: 5.1,
    price: 520,
    rating: 4.6,
    reviews: 2156,
    nextAvailable: "Today, 5:30 PM",
    address: "789 Emergency Way, North District",
    insurance: ["All Major Insurance", "Medicare", "Medicaid"],
    services: ["MRI", "CT Scan", "X-Ray", "Ultrasound", "Emergency Care"],
    urgent: true,
    capacity: "Available",
    waitTime: "10 min"
  },
  {
    id: 4,
    name: "Advanced Medical Imaging",
    type: "Imaging Center",
    distance: 7.8,
    price: 295,
    rating: 4.7,
    reviews: 654,
    nextAvailable: "Feb 8, 10:00 AM",
    address: "321 Diagnostic Lane, East Side",
    insurance: ["Blue Cross", "UnitedHealthcare", "Cigna"],
    services: ["MRI", "CT Scan", "X-Ray"],
    urgent: false,
    capacity: "Available",
    waitTime: "20 min"
  },
  {
    id: 5,
    name: "Community Health Center",
    type: "Clinic",
    distance: 4.2,
    price: 380,
    rating: 4.5,
    reviews: 445,
    nextAvailable: "Feb 7, 2:00 PM",
    address: "555 Community Road, West District",
    insurance: ["Medicare", "Medicaid", "Blue Cross"],
    services: ["MRI", "X-Ray", "Ultrasound"],
    urgent: false,
    capacity: "Limited",
    waitTime: "45 min"
  },
  {
    id: 6,
    name: "University Medical Center",
    type: "Hospital",
    distance: 9.5,
    price: 550,
    rating: 4.9,
    reviews: 3421,
    nextAvailable: "Tomorrow, 11:00 AM",
    address: "888 University Ave, Campus District",
    insurance: ["All Major Insurance"],
    services: ["MRI", "CT Scan", "PET Scan", "X-Ray", "Ultrasound", "Specialty Care"],
    urgent: true,
    capacity: "Available",
    waitTime: "25 min"
  }
];

export function SearchResults({ onSelectFacility }: SearchResultsProps) {
  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-gray-900">
            {facilities.length} facilities found for MRI scan
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
                        {facility.distance} miles away
                      </p>
                      <p className="text-xs text-gray-600">{facility.address}</p>
                    </div>
                  </div>
                  <div className="flex items-start gap-2">
                    <DollarSign className="w-4 h-4 text-gray-400 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        ${facility.price}
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