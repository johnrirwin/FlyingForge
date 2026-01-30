package tagging

import (
	"strings"
)

type Tagger struct {
	rules map[string][]string
}

func New() *Tagger {
	return &Tagger{
		rules: map[string][]string{
			"FAA":         {"faa", "federal aviation", "part 107", "remote id", "airspace"},
			"DJI":         {"dji", "mavic", "phantom", "mini", "air 2", "avata", "inspire"},
			"FPV":         {"fpv", "first person view", "goggles", "betaflight", "freestyle"},
			"Racing":      {"racing", "race", "multiGP", "drone racing league", "drl"},
			"Photography": {"photography", "photo", "camera", "aerial photo", "cinematography"},
			"Videography": {"videography", "video", "footage", "cinematic", "filming"},
			"Commercial":  {"commercial", "enterprise", "industrial", "professional", "business"},
			"Military":    {"military", "defense", "army", "navy", "air force", "warfare"},
			"Delivery":    {"delivery", "package", "logistics", "amazon", "wing", "zipline"},
			"Agriculture": {"agriculture", "farming", "crop", "spray", "agri", "precision ag"},
			"Mapping":     {"mapping", "survey", "lidar", "photogrammetry", "gis", "3d model"},
			"News":        {"news", "announcement", "release", "update", "launch"},
			"Review":      {"review", "test", "hands-on", "comparison", "vs"},
			"Tutorial":    {"tutorial", "how to", "guide", "tips", "learn"},
			"Regulation":  {"regulation", "law", "rule", "policy", "compliance", "legal"},
			"Safety":      {"safety", "crash", "accident", "incident", "hazard"},
			"Technology":  {"technology", "tech", "innovation", "sensor", "battery", "motor"},
			"Autonomous":  {"autonomous", "ai", "machine learning", "obstacle avoidance", "waypoint"},
		},
	}
}

func (t *Tagger) InferTags(title, content string) []string {
	combined := strings.ToLower(title + " " + content)
	tags := make(map[string]bool)

	for tag, keywords := range t.rules {
		for _, keyword := range keywords {
			if strings.Contains(combined, keyword) {
				tags[tag] = true
				break
			}
		}
	}

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}
	return result
}

func (t *Tagger) AddRule(tag string, keywords []string) {
	t.rules[tag] = keywords
}

func (t *Tagger) RemoveRule(tag string) {
	delete(t.rules, tag)
}

func (t *Tagger) GetRules() map[string][]string {
	rules := make(map[string][]string)
	for k, v := range t.rules {
		rules[k] = v
	}
	return rules
}
