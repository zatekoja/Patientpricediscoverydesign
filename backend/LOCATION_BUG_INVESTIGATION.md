# Location Bug Investigation - "Lagos, Lagos" Default Issue

## Issue Summary
**CRITICAL BUG**: Hospital locations are incorrectly defaulting to "Lagos, Lagos" instead of their actual geographic location.

## Root Cause Analysis

### Three Interrelated Problems

#### 1. `inferRegionFromTags()` Function (Line 510-520)
**Problem**: Uses 'lagos' in TAGS to infer region, even when it's metadata, not actual location.

```go
// Current problematic implementation
func (s *ProviderIngestionService) inferRegionFromTags(tags []string) string {
    for _, tag := range tags {
        lower := strings.ToLower(tag)
        if strings.Contains(lower, "lagos") {
            return "Lagos"  // ❌ BUG: Returns Lagos for ANY tag containing "lagos"
        }
        // ... other regions
    }
    return ""
}
```

**Why it's wrong**: A hospital in Port Harcourt might have a 'lagos' tag for metadata/categorization purposes, but this function treats it as the actual location.

#### 2. `buildFallbackGeocodedAddress()` Function (Line 383-390)
**Problem**: Calls `inferRegionFromTags()` to fill missing city, ignoring actual coordinates.

```go
// Current problematic implementation
func (s *ProviderIngestionService) buildFallbackGeocodedAddress(
    facility *entities.Facility, 
    tags []string,
) string {
    if facility.Address.City == "" {
        inferred := s.inferRegionFromTags(tags)  // ❌ Uses tag inference for missing city
        facility.Address.City = inferred         // ❌ Overrides with "Lagos"
    }
    // ...
}
```

**Why it's wrong**: When a facility has coordinates (lat/long) but no city name, instead of using reverse geocoding with the actual coordinates, it falls back to tag-based inference which defaults to "Lagos".

#### 3. `buildGeocodeQuery()` Function (Line 461-507)
**Problem**: Adds inferred region to geocode query, biasing results toward Lagos.

```go
// Current problematic implementation
func (s *ProviderIngestionService) buildGeocodeQuery(
    facility *entities.Facility,
    record providerapi.PriceRecord,
    profile *providerapi.FacilityProfile,
    tags []string,
) string {
    // ... builds query with actual address ...
    
    inferred := s.inferRegionFromTags(tags)  // ❌ Always calls this
    if inferred != "" {
        parts = append(parts, inferred)      // ❌ Adds "Lagos" to query
    }
    
    return strings.Join(parts, ", ")  // Result: "Port Harcourt, Rivers, Nigeria, Lagos"
}
```

**Why it's wrong**: Even when a facility has a correct city and state, if there's a 'lagos' tag, the function appends "Lagos" to the geocoding query, causing the geocoding service to return Lagos coordinates.

## Test Evidence

Created comprehensive test suite in `provider_ingestion_service_location_test.go`:

### Test Results

#### ✅ `TestInferRegionFromTagsProblem` - PASSES (Documents the bug)
- Shows that `inferRegionFromTags(['hospital', 'lagos'])` returns "Lagos"
- Cannot distinguish between metadata tags and actual location
- Logs: `⚠️ KNOWN ISSUE: This function cannot distinguish between 'lagos' as a tag vs actual location`

#### ✅ `TestBuildFallbackGeocodedAddressProblem` - PASSES (Documents the bug)
- Shows that missing city + 'lagos' tag = defaults to Lagos
- Uses tag inference instead of reverse geocoding with coordinates
- Logs: `⚠️ BUG: Missing city + 'lagos' tag = incorrectly defaults to Lagos even with different coordinates`

#### ❌ `TestGeocodeQueryBuildingAddingLagos` - **FAILS** (Proves the bug)
- **Expected**: Query for Port Harcourt hospital should be "Port Harcourt, Rivers, Nigeria"
- **Actual**: Query is "Port Harcourt, Rivers, Nigeria, Lagos"
- **Error**: `❌ BUG: 'lagos' tag should NOT add Lagos to query when city is Port Harcourt`

#### ✅ `TestRootCauseAnalysis` - PASSES (Documents all findings)
- Comprehensive documentation of all three root causes
- Prioritized solution requirements

## Impact

### Affected Scenarios
1. ✅ **Abuja hospitals** - Default to Lagos if tags contain "lagos"
2. ✅ **Port Harcourt hospitals** - Default to Lagos if tags contain "lagos"
3. ✅ **Kano hospitals** - Default to Lagos if tags contain "lagos"
4. ✅ **Ibadan hospitals** - Default to Lagos if tags contain "lagos"
5. ✅ **Any hospital without city** - Defaults to Lagos if tags contain "lagos"

### Correct Behavior
- ✅ **Actual Lagos hospitals** - Should correctly show Lagos (this works)

## Solution Requirements

### Priority 1: Use Coordinates for Reverse Geocoding
When coordinates (latitude, longitude) are available, use reverse geocoding to determine the actual city/region:

```go
if facility.Location.Latitude != 0 && facility.Location.Longitude != 0 {
    // Use reverse geocoding with actual coordinates
    return s.geolocationProvider.ReverseGeocode(ctx, lat, long)
}
```

### Priority 2: Tag Inference as LAST Resort
Only use `inferRegionFromTags()` when:
- No coordinates available
- No address components available
- All other methods exhausted

```go
// ONLY use tag inference as absolute last resort
if facility.Address.City == "" && facility.Location.Latitude == 0 {
    // Try tag inference
    inferred := s.inferRegionFromTags(tags)
}
```

### Priority 3: Don't Add Inferred Region to Geocode Query
Remove the logic that appends inferred region to geocode queries when actual address exists:

```go
// DON'T do this if we have actual address components
// inferred := s.inferRegionFromTags(tags)
// parts = append(parts, inferred)  // ❌ Remove this
```

### Priority 4: Validate Coordinates Match Inferred Region
If using tag inference, validate that inferred region matches coordinates:

```go
inferred := s.inferRegionFromTags(tags)
if !s.coordinatesMatchRegion(facility.Location, inferred) {
    // Coordinates don't match inferred region - don't use it
    return ""
}
```

## Implementation Plan

### Step 1: Fix `buildGeocodeQuery()`
Remove inferred region from geocode queries when actual address exists:

```go
func (s *ProviderIngestionService) buildGeocodeQuery(...) string {
    var parts []string
    
    // Use actual address components ONLY
    if profile != nil && profile.Address.City != "" {
        parts = append(parts, profile.Address.City)
    }
    if profile != nil && profile.Address.State != "" {
        parts = append(parts, profile.Address.State)
    }
    // ... other address components ...
    
    // ❌ REMOVE THIS:
    // inferred := s.inferRegionFromTags(tags)
    // if inferred != "" {
    //     parts = append(parts, inferred)
    // }
    
    return strings.Join(parts, ", ")
}
```

### Step 2: Fix `buildFallbackGeocodedAddress()`
Prioritize reverse geocoding over tag inference:

```go
func (s *ProviderIngestionService) buildFallbackGeocodedAddress(...) string {
    if facility.Address.City == "" {
        // Priority 1: Use reverse geocoding if coordinates available
        if facility.Location.Latitude != 0 && facility.Location.Longitude != 0 {
            result, err := s.geolocationProvider.ReverseGeocode(
                ctx,
                facility.Location.Latitude,
                facility.Location.Longitude,
            )
            if err == nil && result.City != "" {
                facility.Address.City = result.City
                return formatAddress(facility.Address)
            }
        }
        
        // Priority 2: ONLY use tag inference as LAST resort
        inferred := s.inferRegionFromTags(tags)
        if inferred != "" {
            facility.Address.City = inferred
        }
    }
    return formatAddress(facility.Address)
}
```

### Step 3: Add Validation to `inferRegionFromTags()`
Add coordinate validation when available:

```go
func (s *ProviderIngestionService) inferRegionFromTagsSafe(
    tags []string,
    location *entities.Location,
) string {
    inferred := s.inferRegionFromTags(tags)
    
    // If we have coordinates, validate they match the inferred region
    if location != nil && location.Latitude != 0 && inferred != "" {
        if !s.coordinatesMatchRegion(*location, inferred) {
            s.logger.Warn("Inferred region from tags doesn't match coordinates",
                "inferred", inferred,
                "lat", location.Latitude,
                "long", location.Longitude,
            )
            return ""  // Don't use mismatched inference
        }
    }
    
    return inferred
}
```

## Testing Strategy

### Existing Tests (Created)
- ✅ `TestInferRegionFromTagsProblem` - Documents tag inference issue
- ✅ `TestBuildFallbackGeocodedAddressProblem` - Documents fallback issue
- ❌ `TestGeocodeQueryBuildingAddingLagos` - **Currently fails, will pass after fix**
- ✅ `TestRootCauseAnalysis` - Documents complete analysis

### New Tests Needed
1. Test reverse geocoding priority
2. Test coordinate validation
3. Integration test with real provider data

### Verification Plan
1. Fix the three identified functions
2. Run `TestGeocodeQueryBuildingAddingLagos` - should now PASS
3. Test with actual provider data from Abuja, Port Harcourt, Kano hospitals
4. Verify geocoding results match actual hospital locations

## Files Modified
- `provider_ingestion_service.go` - Contains the three problematic functions
- `provider_ingestion_service_location_test.go` - Comprehensive test suite documenting and reproducing the bug

## Next Steps
1. ✅ Investigation complete - Bug confirmed and documented
2. ⏳ Implement fixes for the three functions
3. ⏳ Verify tests pass after fixes
4. ⏳ Integration testing with real data
5. ⏳ Deploy fix to production

## Related Documentation
- See `GEOLOCATION_ROOT_CAUSE_FIX.md` for geolocation system overview
- See `GEOLOCATION_MAPS_IMPLEMENTATION.md` for geocoding implementation
