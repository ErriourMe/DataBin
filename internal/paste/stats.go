package paste

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

const statsKeyPrefix = "paste:stats:"

type DailyView struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type CountryView struct {
	Code  string `json:"code"`
	Count int64  `json:"count"`
}

type Stats struct {
	Views     int64         `json:"views"`
	Countries []CountryView `json:"countries"`
	Daily     []DailyView   `json:"daily"`
}

func statsKey(id string) string {
	return statsKeyPrefix + id
}

func (s *Service) RecordView(ctx context.Context, id, countryCode string) {
	if !idPattern.MatchString(id) {
		return
	}
	key := statsKey(id)
	day := time.Now().UTC().Format("2006-01-02")
	pipe := s.rdb.Pipeline()
	pipe.HIncrBy(ctx, key, "views", 1)
	pipe.HIncrBy(ctx, key, "d:"+day, 1)
	cc := strings.ToUpper(strings.TrimSpace(countryCode))
	if cc != "" && cc != "XX" && cc != "ZZ" {
		pipe.HIncrBy(ctx, key, "c:"+cc, 1)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return
	}
	_ = s.applyStatsTTL(ctx, id)
}

func (s *Service) GetStats(ctx context.Context, id string) (Stats, error) {
	if !idPattern.MatchString(id) {
		return Stats{}, ErrNotFound
	}
	key := statsKey(id)
	all, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return Stats{}, err
	}
	if len(all) == 0 {
		return Stats{}, nil
	}

	var st Stats
	countries := make([]CountryView, 0)
	dailyMap := make(map[string]int64)

	for field, val := range all {
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		switch {
		case field == "views":
			st.Views = n
		case strings.HasPrefix(field, "c:"):
			countries = append(countries, CountryView{Code: strings.TrimPrefix(field, "c:"), Count: n})
		case strings.HasPrefix(field, "d:"):
			dailyMap[strings.TrimPrefix(field, "d:")] = n
		}
	}

	sort.Slice(countries, func(i, j int) bool {
		if countries[i].Count == countries[j].Count {
			return countries[i].Code < countries[j].Code
		}
		return countries[i].Count > countries[j].Count
	})
	st.Countries = countries

	days := make([]string, 0, len(dailyMap))
	for d := range dailyMap {
		days = append(days, d)
	}
	sort.Strings(days)

	st.Daily = make([]DailyView, 0, len(days))
	for _, d := range days {
		st.Daily = append(st.Daily, DailyView{Date: d, Count: dailyMap[d]})
	}
	return st, nil
}

func (s *Service) CheckManageToken(ctx context.Context, id, token string) error {
	if !idPattern.MatchString(id) {
		return ErrNotFound
	}
	metaKey := metaKeyPrefix + id
	exists, err := s.rdb.Exists(ctx, keyPrefix+id).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}
	return checkManageTokenMeta(ctx, s.rdb, metaKey, token)
}

func (s *Service) applyStatsTTL(ctx context.Context, id string) error {
	if s.cfg.TTLHours <= 0 {
		return nil
	}
	ttl := time.Duration(s.cfg.TTLHours) * time.Hour
	return s.rdb.Expire(ctx, statsKey(id), ttl).Err()
}

func (s *Service) deletePasteKeys(ctx context.Context, dataKey, metaKey string) error {
	id := strings.TrimPrefix(dataKey, keyPrefix)
	n, err := s.rdb.Del(ctx, dataKey, metaKey, statsKey(id), burnSeenKey(id)).Result()
	if err != nil {
		return err
	}
	_ = indexRemove(ctx, s.rdb, id)
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
