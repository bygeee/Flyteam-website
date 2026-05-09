package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

type CaptchaEntry struct {
	AnswerHash string
	ExpiresAt  time.Time
	IP         string
	Attempts   int
}

const recruitCaptchaTTL = 180 * time.Second

func (s *Server) captchaHash(token, answer string) string {
	secret := s.cfg.AdminToken
	if secret == "" {
		secret = s.cfg.AdminPassword
	}
	if secret == "" {
		secret = "flyteam-captcha"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token + ":" + strings.ToLower(strings.TrimSpace(answer))))
	return hex.EncodeToString(mac.Sum(nil))
}
func (s *Server) saveRecruitCaptchaCache(token string, entry CaptchaEntry) {
	if s.db == nil {
		return
	}
	payload := map[string]any{
		"answer_hash": entry.AnswerHash,
		"ip":          entry.IP,
		"attempts":    entry.Attempts,
		"expires_at":  entry.ExpiresAt.UTC().Format(time.RFC3339Nano),
	}
	_ = s.saveCacheJSON("captcha", token, payload, entry.ExpiresAt)
}

func (s *Server) loadRecruitCaptchaCache(token string) (CaptchaEntry, bool) {
	var payload struct {
		AnswerHash string `json:"answer_hash"`
		IP         string `json:"ip"`
		Attempts   int    `json:"attempts"`
		ExpiresAt  string `json:"expires_at"`
	}
	if !s.loadCacheJSON("captcha", token, &payload) {
		return CaptchaEntry{}, false
	}
	expiresAt, ok := parseCacheTime(payload.ExpiresAt)
	if !ok {
		return CaptchaEntry{}, false
	}
	return CaptchaEntry{AnswerHash: payload.AnswerHash, IP: payload.IP, Attempts: payload.Attempts, ExpiresAt: expiresAt}, true
}

func (s *Server) cleanupCaptchas() {
	if s.db != nil {
		s.cleanupCache("captcha")
		return
	}
	now := time.Now()
	s.captchaMu.Lock()
	defer s.captchaMu.Unlock()
	for t, c := range s.captchas {
		if now.After(c.ExpiresAt) {
			delete(s.captchas, t)
		}
	}
	if len(s.captchas) > 1000 {
		n := 0
		for t := range s.captchas {
			delete(s.captchas, t)
			n++
			if len(s.captchas) <= 800 || n > 300 {
				break
			}
		}
	}
}
func generateCCodeCaptcha() (string, int) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	kind := r.Intn(6)
	nonce := randomHex(3)
	code := ""
	ans := 0
	switch kind {
	case 0:
		n := r.Intn(4) + 3
		start := r.Intn(3) + 1
		step := r.Intn(3) + 1
		for i := 1; i <= n; i++ {
			ans += start + i*step
		}
		code = fmt.Sprintf(`#include <stdio.h>
int main(void) {
    int s = 0;
    for (int i = 1; i <= %d; i++) {
        s += %d + i * %d;
    }
    printf("%%d", s);
    return 0;
}`, n, start, step)
	case 1:
		n := r.Intn(5) + 5
		mod := []int{2, 3}[r.Intn(2)]
		add := r.Intn(4) + 3
		sub := r.Intn(2) + 1
		base := r.Intn(6) + 12
		ans = base
		for i := 1; i <= n; i++ {
			if i%mod == 0 {
				ans += add
			} else {
				ans -= sub
			}
		}
		code = fmt.Sprintf(`#include <stdio.h>
int main(void) {
    int x = %d;
    for (int i = 1; i <= %d; i++) {
        if (i %% %d == 0) {
            x += %d;
        } else {
            x -= %d;
        }
    }
    printf("%%d", x);
    return 0;
}`, base, n, mod, add, sub)
	case 2:
		x := r.Intn(8) + 3
		y := r.Intn(6) + 2
		if x > y {
			ans = x*2 + y
		} else {
			ans = y*2 - x
		}
		code = fmt.Sprintf(`#include <stdio.h>
int main(void) {
    int x = %d;
    int y = %d;
    if (x > y) {
        x = x * 2 + y;
    } else {
        x = y * 2 - x;
    }
    printf("%%d", x);
    return 0;
}`, x, y)
	case 3:
		n := r.Intn(4) + 3
		a := r.Intn(3) + 1
		ans = a
		for i := 0; i < n; i++ {
			ans += i
		}
		code = fmt.Sprintf(`#include <stdio.h>
int main(void) {
    int a = %d;
    int i = 0;
    while (i < %d) {
        a += i;
        i++;
    }
    printf("%%d", a);
    return 0;
}`, a, n)
	case 4:
		n := r.Intn(4) + 4
		ans = 1
		for i := 1; i <= n; i++ {
			if i%2 == 0 {
				ans += i
			} else {
				ans *= 2
			}
		}
		code = fmt.Sprintf(`#include <stdio.h>
int main(void) {
    int ans = 1;
    for (int i = 1; i <= %d; i++) {
        if (i %% 2 == 0) {
            ans += i;
        } else {
            ans *= 2;
        }
    }
    printf("%%d", ans);
    return 0;
}`, n)
	default:
		a := r.Intn(5) + 2
		b := r.Intn(5) + 2
		c := r.Intn(6) + 1
		if a+b > c {
			ans = (a + b) * c
		} else {
			ans = a + b + c
		}
		code = fmt.Sprintf(`#include <stdio.h>
int main(void) {
    int a = %d;
    int b = %d;
    int c = %d;
    if ((a + b) > c) {
        c = (a + b) * c;
    } else {
        c = a + b + c;
    }
    printf("%%d", c);
    return 0;
}`, a, b, c)
	}
	return "下面 C 语言代码的 printf 输出结果是多少？\n\n/* Flyteam captcha: " + nonce + " */\n" + code, ans
}

func (s *Server) handleRecruitCaptcha(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.checkRateLimit("recruit-captcha:"+ip, 40, 5*time.Minute, true) {
		writeError(w, 429, "验证码刷新过于频繁，请稍后再试。")
		return
	}
	s.cleanupCaptchas()
	challenge, ans := generateCCodeCaptcha()
	token := randomHex(24)
	entry := CaptchaEntry{AnswerHash: s.captchaHash(token, fmt.Sprint(ans)), ExpiresAt: time.Now().Add(recruitCaptchaTTL), IP: ip}
	if s.db != nil {
		s.saveRecruitCaptchaCache(token, entry)
	} else {
		s.captchaMu.Lock()
		s.captchas[token] = entry
		s.captchaMu.Unlock()
	}
	writeJSON(w, 200, map[string]any{"token": token, "challenge": challenge, "expires_in": 180, "captcha_type": "c_output"})
}
func (s *Server) verifyRecruitCaptcha(token, answer, ip string) bool {
	s.cleanupCaptchas()
	token = strings.TrimSpace(token)
	answer = strings.TrimSpace(answer)
	if token == "" || answer == "" {
		return false
	}
	if s.db != nil {
		entry, ok := s.loadRecruitCaptchaCache(token)
		if !ok {
			return false
		}
		s.deleteCache("captcha", token)
		if time.Now().After(entry.ExpiresAt) || entry.IP != ip {
			return false
		}
		expected := entry.AnswerHash
		return hmac.Equal([]byte(expected), []byte(s.captchaHash(token, answer)))
	}
	s.captchaMu.Lock()
	defer s.captchaMu.Unlock()
	entry, ok := s.captchas[token]
	if !ok {
		return false
	}
	if time.Now().After(entry.ExpiresAt) || entry.IP != ip {
		delete(s.captchas, token)
		return false
	}
	entry.Attempts++
	expected := entry.AnswerHash
	ok = hmac.Equal([]byte(expected), []byte(s.captchaHash(token, answer)))
	if ok || entry.Attempts >= 1 {
		delete(s.captchas, token)
	} else {
		s.captchas[token] = entry
	}
	return ok
}
