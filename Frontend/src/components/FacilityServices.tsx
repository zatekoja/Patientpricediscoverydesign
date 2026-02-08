import React, { useState, useEffect, useCallback } from 'react';
import { api } from '../lib/api';
import { ServiceSearchParams, ServiceSearchResponse, FacilityService } from '../types/api';
import { debounce } from 'lodash-es';

interface FacilityServicesProps {
  facilityId: string;
  onServiceSelect?: (service: FacilityService) => void;
  initialSearch?: string;
}

interface ServiceFilters {
  search: string;
  category: string;
  priceRange: [number, number];
  availableOnly: boolean;
  sortBy: 'price' | 'name' | 'category' | 'updated_at';
  sortOrder: 'asc' | 'desc';
}

const INITIAL_FILTERS: ServiceFilters = {
  search: '',
  category: '',
  priceRange: [0, 1000],
  availableOnly: true,
  sortBy: 'price',
  sortOrder: 'asc',
};

export const FacilityServices: React.FC<FacilityServicesProps> = ({
  facilityId,
  onServiceSelect,
  initialSearch = '',
}) => {
  const [services, setServices] = useState<FacilityService[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<ServiceFilters>({
    ...INITIAL_FILTERS,
    search: initialSearch,
  });
  const [pagination, setPagination] = useState({
    currentPage: 1,
    totalPages: 0,
    totalCount: 0,
    pageSize: 20,
    hasNext: false,
    hasPrev: false,
  });

  // TDD-driven search that works across ALL data
  const searchServices = useCallback(async (
    searchFilters: ServiceFilters,
    page: number = 1,
    signal?: AbortSignal
  ) => {
    setLoading(true);
    setError(null);

    try {
      const params: ServiceSearchParams = {
        limit: pagination.pageSize,
        offset: (page - 1) * pagination.pageSize,
        search: searchFilters.search.trim() || undefined,
        category: searchFilters.category || undefined,
        minPrice: searchFilters.priceRange[0] > 0 ? searchFilters.priceRange[0] : undefined,
        maxPrice: searchFilters.priceRange[1] < 1000 ? searchFilters.priceRange[1] : undefined,
        available: searchFilters.availableOnly ? true : undefined,
        sort: searchFilters.sortBy,
        order: searchFilters.sortOrder,
      };

      const response: ServiceSearchResponse = await api.getFacilityServices(
        facilityId,
        params,
        signal
      );

      setServices(response.services);
      setPagination({
        currentPage: response.current_page,
        totalPages: response.total_pages,
        totalCount: response.total_count,
        pageSize: response.page_size,
        hasNext: response.has_next,
        hasPrev: response.has_prev,
      });
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        setError(`Failed to load services: ${err.message}`);
      }
    } finally {
      setLoading(false);
    }
  }, [facilityId, pagination.pageSize]);

  // Debounced search to prevent excessive API calls
  const debouncedSearch = useCallback(
    debounce((searchFilters: ServiceFilters) => {
      searchServices(searchFilters, 1);
    }, 300),
    [searchServices]
  );

  // Initial load and when filters change
  useEffect(() => {
    const abortController = new AbortController();
    searchServices(filters, 1, abortController.signal);
    return () => abortController.abort();
  }, [searchServices, filters]);

  // Handle filter changes
  const handleFilterChange = (newFilters: Partial<ServiceFilters>) => {
    const updatedFilters = { ...filters, ...newFilters };
    setFilters(updatedFilters);
    debouncedSearch(updatedFilters);
  };

  // Handle page changes
  const handlePageChange = (page: number) => {
    if (page >= 1 && page <= pagination.totalPages) {
      searchServices(filters, page);
    }
  };

  // Render service card
  const renderServiceCard = (service: FacilityService) => (
    <div
      key={service.id}
      className={`service-card p-4 border rounded-lg cursor-pointer transition-all ${
        onServiceSelect ? 'hover:shadow-md hover:border-blue-300' : ''
      }`}
      onClick={() => onServiceSelect?.(service)}
    >
      <div className="flex justify-between items-start mb-2">
        <h3 className="font-semibold text-lg text-gray-900">{service.name}</h3>
        <div className="text-right">
          <div className="text-xl font-bold text-blue-600">
            {service.currency} {service.price.toLocaleString()}
          </div>
          {!service.is_available && (
            <span className="text-sm text-red-500">Not Available</span>
          )}
        </div>
      </div>

      <div className="space-y-1 text-sm text-gray-600">
        <div className="flex justify-between">
          <span>Category:</span>
          <span className="capitalize">{service.category}</span>
        </div>
        {service.code && (
          <div className="flex justify-between">
            <span>Code:</span>
            <span>{service.code}</span>
          </div>
        )}
        <div className="flex justify-between">
          <span>Duration:</span>
          <span>{service.estimated_duration} mins</span>
        </div>
        {service.description && (
          <p className="text-gray-700 mt-2">{service.description}</p>
        )}
      </div>
    </div>
  );

  // Render pagination controls
  const renderPagination = () => {
    if (pagination.totalPages <= 1) return null;

    const pages = [];
    const maxVisible = 5;
    const startPage = Math.max(1, pagination.currentPage - Math.floor(maxVisible / 2));
    const endPage = Math.min(pagination.totalPages, startPage + maxVisible - 1);

    // Previous button
    pages.push(
      <button
        key="prev"
        onClick={() => handlePageChange(pagination.currentPage - 1)}
        disabled={!pagination.hasPrev}
        className="px-3 py-2 border rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
      >
        Previous
      </button>
    );

    // Page numbers
    for (let i = startPage; i <= endPage; i++) {
      pages.push(
        <button
          key={i}
          onClick={() => handlePageChange(i)}
          className={`px-3 py-2 border rounded ${
            i === pagination.currentPage
              ? 'bg-blue-500 text-white border-blue-500'
              : 'hover:bg-gray-50'
          }`}
        >
          {i}
        </button>
      );
    }

    // Next button
    pages.push(
      <button
        key="next"
        onClick={() => handlePageChange(pagination.currentPage + 1)}
        disabled={!pagination.hasNext}
        className="px-3 py-2 border rounded disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50"
      >
        Next
      </button>
    );

    return (
      <div className="flex justify-center items-center space-x-2 mt-6">
        {pages}
      </div>
    );
  };

  return (
    <div className="facility-services">
      {/* Search and Filters */}
      <div className="mb-6">
        <div className="flex flex-col lg:flex-row gap-4">
          {/* Search Input */}
          <div className="flex-1">
            <input
              type="text"
              placeholder="Search procedures (searches across all data)..."
              value={filters.search}
              onChange={(e) => handleFilterChange({ search: e.target.value })}
              className="w-full px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          {/* Category Filter */}
          <select
            value={filters.category}
            onChange={(e) => handleFilterChange({ category: e.target.value })}
            className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            <option value="">All Categories</option>
            <option value="diagnostic">Diagnostic</option>
            <option value="therapeutic">Therapeutic</option>
            <option value="surgical">Surgical</option>
            <option value="preventive">Preventive</option>
          </select>

          {/* Sort Options */}
          <select
            value={`${filters.sortBy}-${filters.sortOrder}`}
            onChange={(e) => {
              const [sortBy, sortOrder] = e.target.value.split('-') as [typeof filters.sortBy, typeof filters.sortOrder];
              handleFilterChange({ sortBy, sortOrder });
            }}
            className="px-4 py-2 border rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            <option value="price-asc">Price: Low to High</option>
            <option value="price-desc">Price: High to Low</option>
            <option value="name-asc">Name: A to Z</option>
            <option value="name-desc">Name: Z to A</option>
            <option value="category-asc">Category</option>
          </select>
        </div>

        {/* Availability Toggle */}
        <div className="mt-4">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={filters.availableOnly}
              onChange={(e) => handleFilterChange({ availableOnly: e.target.checked })}
              className="mr-2"
            />
            <span>Show available services only</span>
          </label>
        </div>
      </div>

      {/* Results Info */}
      <div className="mb-4 text-sm text-gray-600">
        {pagination.totalCount > 0 ? (
          <span>
            Showing {((pagination.currentPage - 1) * pagination.pageSize) + 1} to{' '}
            {Math.min(pagination.currentPage * pagination.pageSize, pagination.totalCount)} of{' '}
            <strong>{pagination.totalCount}</strong> services
          </span>
        ) : (
          <span>No services found</span>
        )}
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {/* Loading State */}
      {loading && (
        <div className="flex justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      )}

      {/* Services Grid */}
      {!loading && services.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {services.map(renderServiceCard)}
        </div>
      )}

      {/* Empty State */}
      {!loading && services.length === 0 && !error && (
        <div className="text-center py-12 text-gray-500">
          <div className="text-lg mb-2">No services found</div>
          <div>Try adjusting your search criteria or filters</div>
        </div>
      )}

      {/* Pagination */}
      {renderPagination()}
    </div>
  );
};

export default FacilityServices;
