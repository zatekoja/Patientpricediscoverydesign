package services

import (
	"math"
	"sort"
	"strings"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

type ScoredResult struct {
	Facility       *entities.Facility
	Score          float64
	ScoreBreakdown map[string]float64
}

type SearchRankingService struct {
	wLexical   float64
	wConcept   float64
	wGeo       float64
	wSpecialty float64
}

func NewSearchRankingService() *SearchRankingService {
	return &SearchRankingService{
		wLexical:   0.3,
		wConcept:   0.3,
		wGeo:       0.2,
		wSpecialty: 0.2,
	}
}

func (s *SearchRankingService) Rank(facilities []*entities.Facility, interp *QueryInterpretation, userLat, userLon float64) []ScoredResult {
	if len(facilities) == 0 {
		return nil
	}

	scored := make([]ScoredResult, len(facilities))
	for i, f := range facilities {
		score, breakdown := s.calculateScore(f, interp, userLat, userLon)
		scored[i] = ScoredResult{
			Facility:       f,
			Score:          score,
			ScoreBreakdown: breakdown,
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	return scored
}

func (s *SearchRankingService) calculateScore(f *entities.Facility, interp *QueryInterpretation, lat, lon float64) (float64, map[string]float64) {
	breakdown := make(map[string]float64)

	// 1. Lexical Match
	lexScore := 0.0
	if interp != nil {
		q := strings.ToLower(interp.OriginalQuery)
		if q != "" {
			if strings.Contains(strings.ToLower(f.Name), q) {
				lexScore += 1.0
			}
			for _, tag := range f.Tags {
				if strings.Contains(strings.ToLower(tag), q) {
					lexScore += 0.5
				}
			}
			if lexScore > 1.0 {
				lexScore = 1.0
			}
		}
	}
	breakdown["lexical"] = lexScore * s.wLexical

	// 2. Concept Overlap (Placeholder as entity lacks concept fields)
	conceptScore := 0.0
	// Implementation note: If we updated Entity to include Concepts, we would match interp.MappedConcepts here.
	breakdown["concept"] = conceptScore * s.wConcept

	// 3. Geo Proximity
	geoScore := 0.0
	if lat != 0 && lon != 0 && f.Location.Latitude != 0 && f.Location.Longitude != 0 {
		dist := distance(lat, lon, f.Location.Latitude, f.Location.Longitude)
		// decay score: 1.0 at 0km, 0.5 at 10km
		geoScore = 1.0 / (1.0 + dist/10.0)
	}
	breakdown["geo"] = geoScore * s.wGeo

	total := breakdown["lexical"] + breakdown["concept"] + breakdown["geo"]
	return total, breakdown
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	p := 0.017453292519943295
	a := 0.5 - math.Cos((lat2-lat1)*p)/2 + math.Cos(lat1*p)*math.Cos(lat2*p)*(1-math.Cos((lon2-lon1)*p))/2
	return 12742 * math.Asin(math.Sqrt(a))
}
