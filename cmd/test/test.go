package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

var (
	K    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	DB   int
	DM   int
	DV   int
	FV   float64
	F1   int
	F2   int
	o    map[uint8]int
	ZERO *TiktokT
	ONE  *TiktokT
)

func Init() {
	DB = 28
	DM = 268435455
	DV = 1 << 28
	FV = math.Pow(2, 52)
	F1 = 24
	F2 = 4
	o = make(map[uint8]int, 128)
	for i := '0'; i <= '9'; i++ {
		o[uint8(i)] = int(i - '0')
	}
	for i := 'a'; i <= 'z'; i++ {
		o[uint8(i)] = int(i - 'a' + 10)
	}
	for i := 'A'; i <= 'Z'; i++ {
		o[uint8(i)] = int(i - 'A' + 10)
	}
	ZERO = &TiktokT{T: 0, S: 0, Values: map[int]int{}}
	ONE = &TiktokT{S: 0, T: 1, Values: map[int]int{0: 1}}
}

type TiktokT struct {
	Values map[int]int
	T      int
	S      int
}

type TiktokR struct {
	M   *TiktokT
	Mp  int
	Mpl int
	Mph int
	Um  int
	Mt2 int
}

type TiktokF struct {
	CoEff *TiktokT
	D     *TiktokT
	Dmp1  *TiktokT
	Dmq1  *TiktokT
	e     int
	n     *TiktokT
	p     *TiktokT
	q     *TiktokT
}

func (this *TiktokT) clamp() {
	e := this.S & DM
	for this.T > 0 && this.Values[this.T-1] == e {
		this.T--
	}
}

func (this *TiktokT) subTo(e *TiktokT, t *TiktokT) {
	n := int(math.Min(float64(e.T), float64(this.T)))
	r := 0
	i := 0
	for i < n {
		r += this.Values[i] - e.Values[i]
		t.Values[i] = r & DM
		i++
		r >>= DB
	}
	if e.T < this.T {
		r -= e.S
		for i < this.T {
			r += this.Values[i]
			t.Values[i] = r & DM
			i++
			r >>= DB
			r += this.S
		}
	} else {
		r += this.S
		for i < e.T {
			r -= e.Values[i]
			t.Values[i] = r & DM
			i++
			r >>= DB
		}
		r -= e.S
	}
	if r < 0 {
		t.S = -1
	} else {
		t.S = 0
	}
	if r < -1 {
		t.Values[i] = DV + r
		i++
	} else {
		if r > 0 {
			t.Values[i] = r
			i++
		}
	}
	t.T = i
	t.clamp()
}
func (this *TiktokT) fromInt(e int) {
	this.T = 1
	if e < 0 {
		this.S = -1
	} else {
		this.S = 0
	}
	if e > 0 {
		this.Values[0] = e
	} else if e < -1 {
		this.Values[0] = e + DV
	} else {
		this.T = 0
	}
}
func (this *TiktokT) isEven() bool {
	i := 0
	if this.T > 0 {
		i = 1 & this.Values[0]
	} else {
		i = this.S
	}
	return 0 == i
}
func n() *TiktokT {
	return newTiktokT(nil, nil, nil)
}

func p(e int) *TiktokT {
	var t = n()
	t.fromInt(e)
	return t
}
func (this *TiktokT) invDigit() int {
	if this.T < 1 {
		return 0
	}
	e := this.Values[0]
	if 0 == (1 & e) {
		return 0
	}
	t := e & 3
	t = (t * (2 - (15&e)*t) & 15)
	t = t * (2 - (255&e)*t) & 255
	t = t * (2 - ((65535 & e) * t & 65535)) & 65535
	t = t * (2 - e*t%DV) % DV
	if t > 0 {
		return DV - t
	} else {
		return -t
	}
}

func newD(e *TiktokT) *TiktokR {
	a := TiktokR{}
	a.M = e
	return &a
}
func newB(e *TiktokT) *TiktokR {
	a := TiktokR{}
	a.M = e
	a.Mp = e.invDigit()
	a.Mpl = 32767 & a.Mp
	a.Mph = a.Mp >> 15
	a.Um = (1 << (DB - 15)) - 1
	a.Mt2 = 2 * e.T
	return &a
}

func newE(e *TiktokT) *TiktokR {
	a := TiktokR{}
	//a.r2 = n(),
	//a.q3 = n()
	//T.ONE.dlShiftTo(2 * e.T, this.r2),
	//this.mu = this.r2.divide(e),
	//this.M = e
	return &a
}
func (this *TiktokT) negate() *TiktokT {
	var e = n()
	ZERO.subTo(this, e)
	return e

}
func (this *TiktokT) abs() *TiktokT {
	if this.S < 0 {
		return this.negate()
	}
	return this
}
func (this *TiktokT) dlShiftTo(e int, t *TiktokT) {
	for i := this.T - 1; i >= 0; i-- {
		t.Values[i+e] = this.Values[i]
	}
	for i := e - 1; i >= 0; i-- {
		t.Values[i] = 0
	}
	t.T = this.T + e
	t.S = this.S
}
func (this *TiktokT) copyTo(e *TiktokT) {
	for t := this.T - 1; t >= 0; t-- {
		e.Values[t] = this.Values[t]
	}
	e.T = this.T
	e.S = this.S
}

func (this *TiktokT) lShiftTo(e int, t *TiktokT) {
	r := e % DB
	n := DB - r
	a := (1 << n) - 1
	s := int(math.Floor(float64(e) / float64(DB)))
	o := this.S << r & DM
	for i := this.T - 1; i >= 0; i-- {
		t.Values[i+s+1] = this.Values[i]>>n | o
		o = (this.Values[i] & a) << r
	}
	for i := s - 1; i >= 0; i-- {
		t.Values[i] = 0
	}
	t.Values[s] = o
	t.T = this.T + s + 1
	t.S = this.S
	t.clamp()
}
func (this *TiktokT) divRemTo(e *TiktokT, i *TiktokT, r *TiktokT) {
	a := e.abs()
	if !(a.T <= 0) {
		s := this.abs()
		if s.T < a.T {
			if i != nil {
				i.fromInt(0)
			}
			if r != nil {
				this.copyTo(r)
			}
			return
		}
		if r == nil {
			r = n()
		}
		o := n()
		l := this.S
		u := e.S
		c := DB - f(a.Values[a.T-1])
		if c > 0 {
			a.lShiftTo(c, o)
			s.lShiftTo(c, r)
		} else {
			a.copyTo(o)
			s.copyTo(r)
		}
		h := o.T
		p := o.Values[h-1]
		if 0 != p {
			d := p * (1 << F1)
			if h > 1 {
				d = d + o.Values[h-2]>>F2
			}
			b := float64(FV / float64(d))
			v1 := 1 << F1
			v := float64(float64(v1) / float64(d))
			g := 1 << F2
			y := r.T
			m := y - h
			O := i
			if i == nil {
				O = n()
			}
			o.dlShiftTo(m, O)
			if r.compareTo(O) >= 0 {
				r.Values[r.T] = 1
				r.T++
				r.subTo(O, r)
			}
			ONE.dlShiftTo(h, O)
			O.subTo(o, o)
			for o.T < h {
				o.Values[o.T] = 0
				o.T++
			}

			for m--; m >= 0; m-- {
				S := DM
				y--
				if r.Values[y] != p {
					S = int(math.Floor(float64(r.Values[y])*b + float64(r.Values[y-1]+g)*v))
				}
				r.Values[y] += o.am(0, S, r, m, 0, h)
				if r.Values[y] < S {
					o.dlShiftTo(m, O)
					r.subTo(O, r)
					for S--; r.Values[y] < S; S-- {
						r.subTo(O, r)
					}
				}
			}
			if nil != i {
				r.drShiftTo(h, i)
				if l != u {
					ZERO.subTo(i, i)
				}
			}
			r.T = h
			r.clamp()
			if c > 0 {
				r.rShiftTo(c, r)
			}
			if l < 0 {
				ZERO.subTo(r, r)
			}
		}
	}
}
func (this *TiktokR) convert(e *TiktokT) *TiktokT {
	i := n()
	e.abs().dlShiftTo(this.M.T, i)
	i.divRemTo(this.M, nil, i)
	if e.S < 0 {
		if i.compareTo(ZERO) > 0 {
			this.M.subTo(i, i)
		}
	}
	return i
}
func (this *TiktokT) mod(e *TiktokT) *TiktokT {
	i := n()
	this.abs().divRemTo(e, nil, i)
	if this.S < 0 {
		if i.compareTo(ZERO) > 0 {
			e.subTo(i, i)
		}
	}
	return i
}
func (this *TiktokT) drShiftTo(e int, t *TiktokT) {
	for i := e; i < this.T; i++ {
		t.Values[i-e] = this.Values[i]
	}
	t.T = int(math.Max(float64(this.T-e), float64(0)))
	t.S = this.S
}
func (this *TiktokT) squareTo(e *TiktokT) {
	t := this.abs()
	e.T = 2 * t.T
	i := e.T
	for i--; i >= 0; i-- {
		e.Values[i] = 0
	}
	for i = 0; i < t.T-1; i++ {
		var r = t.am(i, t.Values[i], e, 2*i, 0, 1)
		e.Values[i+t.T] += t.am(i+1, 2*t.Values[i], e, 2*i+1, r, t.T-i-1)
		if e.Values[i+t.T] >= DV {
			e.Values[i+t.T] -= DV
			e.Values[i+t.T+1] = 1
		}
	}
	if e.T > 0 {
		e.Values[e.T-1] += t.am(i, t.Values[i], e, 2*i, 0, 1)
	}
	e.S = 0
	e.clamp()
}
func (this *TiktokR) reduce(e *TiktokT) {
	for e.T <= this.Mt2 {
		e.Values[e.T] = 0
		e.T++
	}
	for t := 0; t < this.M.T; t++ {
		i := 32767 & e.Values[t]
		r := (i*this.Mpl + ((i*this.Mph+(e.Values[t]>>15)*this.Mpl&this.Um)<<15))&DM
		i = t + this.M.T
		e.Values[i] += this.M.am(0, r, e, t, 0, this.M.T)
		for e.Values[i] >= DV {
			e.Values[i] -= DV
			i++
			e.Values[i]++
		}
	}
	e.clamp()
	e.drShiftTo(this.M.T, e)
	if e.compareTo(this.M) >= 0 {
		e.subTo(this.M, e)
	}
}

func (this *TiktokR) sqrTo(e, t *TiktokT) {
	e.squareTo(t)
	this.reduce(t)
}

func (this *TiktokR) mulTo(e, t, i *TiktokT) {
	e.multiplyTo(t, i)
	this.reduce(i)
}
func (this *TiktokR) revert(e *TiktokT) *TiktokT {
	var t = n()
	e.copyTo(t)
	this.reduce(t)
	return t
}

func (this *TiktokT) modPow(e, t *TiktokT) *TiktokT {
	i := 0
	var r *TiktokR
	a := e.bitLength()
	s := p(1)
	if a <= 0 {
		return s
	}
	if a < 18 {
		i = 1
	} else if a < 48 {
		i = 3
	} else if a < 144 {
		i = 4
	} else if a < 765 {
		i = 5
	} else {
		i = 6
	}
	if a < 8 {
		r = newD(t)
	} else if t.isEven() {
		r = newE(t)
	} else {
		r = newB(t)
	}

	l := 3
	u := i - 1
	c := (1 << i) - 1
	o := make([]*TiktokT, c+1)
	o[1] = r.convert(this)
	if i > 1 {
		h := n()
		for r.sqrTo(o[1], h); l <= c; {
			o[l] = n()
			r.mulTo(h, o[l-2], o[l])
			l += 2
		}
	}
	y := e.T - 1
	m := true
	O := n()
	for a = f(e.Values[y]) - 1; y >= 0; {
		v := 0
		if a >= u {
			v = e.Values[y] >> (a - u)&c

		} else {
			v = (e.Values[y] & ((1<<(a+1) - 1)) << (u - a))
			if y > 0 {
				v |= e.Values[y-1] >> (DB + a - u)
			}
		}
		l = i
		for 0 == (1 & v) {
			v >>= 1
			l--
		}
		a -= l
		if a < 0 {
			a += DB
			y--
		}
		if m {
			o[v].copyTo(s)
			m = false
		} else {
			for l > 1 {
				r.sqrTo(s, O)
				r.sqrTo(O, s)
				l -= 2
			}
			if l > 0 {
				r.sqrTo(s, O)
			} else {
				g := s
				s = O
				O = g
			}
			r.mulTo(O, o[v], s)
		}
		for y >= 0 && 0 == (e.Values[y] & ( 1 << a )) {
			r.sqrTo(s, O)
			g := s
			s = O
			O = g
			a--
			if a < 0 {
				a = DB - 1
				y--
			}
		}

	}

	return r.revert(s)
}
func f(e int) int {
	i := 1
	if t := e >> 16; t != 0 {
		e = t
		i += 16
	}
	if t := e >> 8; t != 0 {
		e = t
		i += 8
	}
	if t := e >> 4; t != 0 {
		e = t
		i += 4
	}
	if t := e >> 2; t != 0 {
		e = t
		i += 2
	}
	if t := e >> 1; t != 0 {
		e = t
		i += 1
	}
	return i
}

func (this *TiktokT) bitLength() int {
	if this.T <= 0 {
		return 0
	} else {
		return DB*(this.T-1) + f(this.Values[this.T-1]^this.S&DM)
	}
}

func c(e string, i int) int {
	a := e[i]
	var k, check = o[a]
	if !check {
		return -1
	}
	return k
}
func (this *TiktokT) compareTo(e *TiktokT) int {
	var t = this.S - e.S
	if 0 != t {
		return t
	}
	var i = this.T
	t = i - e.T
	if 0 != t {
		if this.S < 0 {
			return -t
		} else {
			return t
		}
	}
	for i--; i >= 0; i-- {
		t = this.Values[i] - e.Values[i]
		if 0 != t {
			return t
		}
	}
	return 0
}
func (this *TiktokT) addTo(e *TiktokT, t *TiktokT) {
	i := 0
	r := 0
	n := int(math.Min(float64(e.T), float64(this.T)))
	for i < n {
		r += this.Values[i] + e.Values[i]
		t.Values[i] = r & DM
		i++
		r >>= DB
	}
	if e.T < this.T {
		r += e.S
		for i < this.T {
			r += this.Values[i]
			t.Values[i] = r & DM
			i++
			r >>= DB
		}
		r += this.S
	} else {
		r += this.S
		for i < e.T {
			r += e.Values[i]
			t.Values[i] = r & DM
			i++
			r >>= DB
		}
		r += e.S
	}
	if r < 0 {
		t.S = -1
	} else {
		t.S = 0
	}
	if r > 0 {
		t.Values[i] = r
		i++
	} else if r < -1 {
		t.Values[i] = DV + r
		i++
	}
	t.T = i
	t.clamp()
}
func (this *TiktokT) am(e int, t int, i *TiktokT, r int, n int, a int) int {
	s := 16383 & t
	o := t >> 14
	for a--; a >= 0; a-- {
		l := 16383 & this.Values[e]
		u := this.Values[e] >> 14
		e++
		c := o*l + u*s
		l = s*l + ((16383 & c) << 14) + i.Values[r] + n
		n = (l >> 28) + (c >> 14) + o*u
		i.Values[r] = 268435455 & l
		r++
	}
	return n
}
func (this *TiktokT) multiplyTo(e *TiktokT, i *TiktokT) {
	r := this.abs()
	n := e.abs()
	a := r.T
	i.T = a + n.T
	for a--; a >= 0; a-- {
		i.Values[a] = 0
	}
	for a = 0; a < n.T; a++ {
		i.Values[a+r.T] = r.am(0, n.Values[a], i, a, 0, r.T)
	}
	i.S = 0
	i.clamp()
	if this.S != e.S {
		ZERO.subTo(i, i)
	}
}

func (this *TiktokT) add(e *TiktokT) *TiktokT {
	var t = n()
	this.addTo(e, t)
	return t
}
func (this *TiktokT) rShiftTo(e int, t *TiktokT) {
	t.S = this.S
	i := int(math.Floor(float64(e) / float64(DB)))
	if i >= this.T {
		t.T = 0
	} else {
		r := e % DB
		n := DB - r
		a := (1 << r) - 1
		t.Values[0] = this.Values[i] >> r
		for s := i + 1; s < this.T; s++ {
			t.Values[s-i-1] |= (this.Values[s] & a) << n
			t.Values[s-i] = this.Values[s] >> r
		}
		if r > 0 {
			t.Values[this.T-i-1] |= (this.S & a) << n
		}
		t.T = this.T - i
		t.clamp()
	}
}
func (this *TiktokT) subtract(e *TiktokT) *TiktokT {
	var t = n()
	this.subTo(e, t)
	return t
}
func (this *TiktokT) multiply(e *TiktokT) *TiktokT {
	var t = n()
	this.multiplyTo(e, t)
	return t
}

//func (t *TiktokT) Even() bool {
//	if t.T > 0 {
//		return 0 == (1 & t.Values[0])
//	} else {
//		return 0 == t.S
//	}
//}
func (t *TiktokT) claim() {
	e := t.S & DM
	for t.T > 0 && t.Values[t.T-1] == e {
		t.T--
	}
}
func l(e int) string {
	chars := "0123456789abcdefghijklmnopqrstuvwxyz"
	return string(chars[e])
}
func V(e string) string {
	var t, i, v int
	r := ""
	n := 0

	for t = 0; t < len(e) && "=" != string(e[t]); t++ {
		v = strings.IndexByte(K, e[t])
		if v < 0 {
			continue
		}
		if 0 == n {
			r += string(l(v >> 2))
			i = 3 & v
			n = 1
		} else if 1 == n {
			r += string(l(i<<2 | v>>4))
			i = 15 & v
			n = 2
		} else if 2 == n {
			r += string(l(i))
			r += string(l(v >> 2))
			i = 3 & v
			n = 3
		} else {
			r += string(l(i<<2 | v>>4))
			r += string(l(15 & v))
			n = 0
		}
	}

	if 1 == n {
		r += string(l(i << 2))
	}
	return r
}

func newTiktokT(e interface{}, t interface{}, i interface{}) *TiktokT {
	this := &TiktokT{Values: map[int]int{}}
	if e != nil {
		//if num, ok := e.(int); ok {
		//	//this.fromNumber(num, T, i)
		//} else
		if str, ok := e.(string); t == nil && !ok {
			this.fromString(str, 256)
		} else {
			this.fromString(str, t.(int))
		}
	}
	return this
}

func converJsonToTiktokT(json map[string]interface{}) *TiktokT {
	data := TiktokT{Values: map[int]int{}}
	for key, value := range json {
		index, err := strconv.Atoi(key)
		if err == nil {
			data.Values[index] = int(value.(float64))
		} else if key == "S" {
			data.S = int(value.(float64))
		} else if key == "T" {
			data.T = int(value.(float64))
		} else {
			fmt.Println("Error when converJsonToTiktokT", key, value)
		}
	}
	return &data
}
func getKey() (tiktokF *TiktokF) {
	key := []byte("{\"n\":{\"0\":246906281,\"1\":119423612,\"2\":184470917,\"3\":2922993,\"4\":170660920,\"5\":42442176,\"6\":198614453,\"7\":71022774,\"8\":54880240,\"9\":113961625,\"10\":153151916,\"11\":9629143,\"12\":49991276,\"13\":93293663,\"14\":128083550,\"15\":175894062,\"16\":173863116,\"17\":98070484,\"18\":33375133,\"19\":74145078,\"20\":221772946,\"21\":148216763,\"22\":65142841,\"23\":178344970,\"24\":177764036,\"25\":199455663,\"26\":152502742,\"27\":53581994,\"28\":50492887,\"29\":154255101,\"30\":86285634,\"31\":537057,\"32\":185045239,\"33\":237522015,\"34\":214768191,\"35\":76408052,\"36\":54466,\"T\":37,\"S\":0},\"e\":65537,\"d\":{\"0\":184109361,\"1\":200785900,\"2\":49809820,\"3\":38014117,\"4\":148981482,\"5\":241763857,\"6\":264266389,\"7\":30235496,\"8\":158132663,\"9\":15516504,\"10\":133026790,\"11\":68372805,\"12\":252530114,\"13\":259214850,\"14\":237800341,\"15\":248699245,\"16\":209449198,\"17\":106845594,\"18\":116380518,\"19\":192236943,\"20\":85636374,\"21\":185705764,\"22\":192340738,\"23\":241698716,\"24\":48997886,\"25\":253578869,\"26\":231297904,\"27\":267732611,\"28\":250857781,\"29\":31078656,\"30\":146482456,\"31\":8090102,\"32\":224560142,\"33\":175235333,\"34\":84160460,\"35\":171324533,\"36\":3386,\"T\":37,\"S\":0},\"p\":{\"0\":90627037,\"1\":207939090,\"2\":918320,\"3\":1751102,\"4\":137533835,\"5\":103008615,\"6\":6054504,\"7\":2524561,\"8\":74456901,\"9\":81923578,\"10\":266292657,\"11\":126746409,\"12\":153972934,\"13\":45255375,\"14\":116477608,\"15\":103267229,\"16\":137874091,\"17\":76817976,\"18\":246,\"T\":19,\"S\":0},\"q\":{\"0\":265085501,\"1\":240870091,\"2\":184884146,\"3\":177652101,\"4\":34840647,\"5\":161277849,\"6\":73121251,\"7\":149929623,\"8\":107019720,\"9\":239786212,\"10\":157551599,\"11\":178930206,\"12\":67369047,\"13\":106939853,\"14\":33304091,\"15\":202627601,\"16\":119461675,\"17\":40372469,\"18\":221,\"T\":19,\"S\":0},\"dmp1\":{\"0\":265093329,\"1\":43221792,\"2\":33051786,\"3\":50608845,\"4\":191997617,\"5\":62105896,\"6\":238213946,\"7\":37070613,\"8\":116653622,\"9\":240542630,\"10\":12132364,\"11\":100432323,\"12\":162753181,\"13\":225828835,\"14\":216063915,\"15\":247455024,\"16\":113424070,\"17\":156716115,\"18\":46,\"T\":19,\"S\":0},\"dmq1\":{\"0\":99285389,\"1\":220495715,\"2\":175580664,\"3\":241820165,\"4\":104519361,\"5\":188160802,\"6\":157824899,\"7\":129007211,\"8\":184289861,\"9\":258501737,\"10\":210841585,\"11\":100717985,\"12\":157467759,\"13\":170540647,\"14\":171224940,\"15\":149972032,\"16\":39927112,\"17\":96310974,\"18\":54,\"T\":19,\"S\":0},\"coeff\":{\"0\":164347554,\"1\":13991341,\"2\":14552478,\"3\":246236067,\"4\":112335125,\"5\":9716711,\"6\":234117532,\"7\":54909736,\"8\":192459404,\"9\":16836582,\"10\":144824938,\"11\":33422002,\"12\":149464469,\"13\":234416822,\"14\":231234041,\"15\":177193671,\"16\":143371500,\"17\":182894617,\"18\":111,\"T\":19,\"S\":0}}")
	data := map[string]interface{}{}
	json.Unmarshal(key, &data)
	coeffData := data["coeff"].(map[string]interface{})
	dData := data["d"].(map[string]interface{})
	dmp1Data := data["dmp1"].(map[string]interface{})
	dmq1Data := data["dmq1"].(map[string]interface{})
	eData := int(data["e"].(float64))
	nData := data["n"].(map[string]interface{})
	pData := data["p"].(map[string]interface{})
	qData := data["q"].(map[string]interface{})
	tF := TiktokF{e: eData}
	tF.CoEff = converJsonToTiktokT(coeffData)
	tF.D = converJsonToTiktokT(dData)
	tF.Dmp1 = converJsonToTiktokT(dmp1Data)
	tF.Dmq1 = converJsonToTiktokT(dmq1Data)
	tF.n = converJsonToTiktokT(nData)
	tF.p = converJsonToTiktokT(pData)
	tF.q = converJsonToTiktokT(qData)
	return &tF
}

func (this *TiktokF) doPrivate(e *TiktokT) *TiktokT {
	if this.p == nil || this.q == nil {
		return e.modPow(this.D, this.n)
	}
	var t = e.mod(this.p).modPow(this.Dmp1, this.p)
	i := e.mod(this.q).modPow(this.Dmq1, this.q)
	for t.compareTo(i) < 0 {
		t = t.add(this.p)
	}
	return t.subtract(i).multiply(this.CoEff).mod(this.p).multiply(this.q).add(i)
}

func (this *TiktokT) toByteArray() []int {
	e := this.T
	t := make([]int,127)
	t[0] = this.S
	r := DB - e*DB%8
	n := 0
	i := 0
	if e > 0 {
		e--
		if r < DB {
			i = this.Values[e] >> r
			if i != (this.S&DM)>>r {
				t[n] = i | this.S<<(DB-r)
				n++
			}
		}
		for e >= 0 {
			if r < 8 {
				i = (this.Values[e] & ((1 << r )-1)) << (8 - r)
				e--
				r += DB - 8
				i |= this.Values[e] >> r
			} else {
				r -= 8
				i = (this.Values[e] >> r) & 255
				if r <= 0 {
					r += DB
					e--
				}
			}
			if 0 != (128 & i) {
				i |= -256
			}
			if 0 == n {
				if (128 & this.S) != (128 & i) {
					n++
				}
			}
			if n > 0 || i != this.S {
				t[n] = i
				n++
			}
		}
	}
	return t
}

func (this *TiktokT) fromString(e string, i int) {
	var r int
	if i == 16 {
		r = 4
	} else if i == 8 {
		r = 3
	} else if i == 256 {
		r = 8
	} else if i == 2 {
		r = 1
	} else if i == 32 {
		r = 5
	} else {
		//if (4 != i) return void this.fromRadix(e, i);
		r = 2
	}
	this.T = 0
	this.S = 0
	a := false
	s := 0

	for n := len(e) - 1; n >= 0; n-- {
		var o int
		if r == 8 {
			o = int(255 & e[n])
		} else {
			o = c(e, n)
		}
		if o < 0 {
			if e[n] == '-' {
				a = true
			}
		} else {
			a = false
			if 0 == s {
				this.Values[this.T] = o
				this.T++
			} else if s+r > DB {
				this.Values[this.T-1] |= (o & ((1 << (DB - s)) - 1)) << s
				this.T++
				this.Values[this.T-1] = o >> (DB - s)
			} else {
				this.Values[this.T-1] |= o << s
			}
			s += r
			if s >= DB {
				s -= DB
			}
		}
	}
	this.claim()
	if a {
		ZERO.subTo(this, this)
	}
}
func L(e string, i int) *TiktokT {
	return newTiktokT(e, i, nil)
}
func decryptReturn(e *TiktokT, t int) string {
	var i = e.toByteArray()
	r := 0
	for r < len(i) && 0 == i[r] {
		r++
	}
	if len(i)-r != t-1 || 2 != i[r] {
		return ""
	}
	for r++; 0 != i[r]; {
		r++
		if r >= len(i) {
			return ""
		}
	}
	n := ""
	for r < len(i) {
		a := 255 & i[r]
		if a < 128 {
			n += string(a)
		} else if a > 191 && a < 224 {
			n += string((31&a)<<6 | 63&i[r+1])
			r++
		} else {
			n += string((15&a)<<12 | (63&i[r+1])<<6 | 63&i[r+2])
			r += 2
		}
		r++
	}
	return n
}
func (this *TiktokF) Decrypt(e string) string {
	t := L(e, 16)
	i := this.doPrivate(t)
	if nil == i {
		return ""
	} else {
		return decryptReturn(i, (this.n.bitLength()+7)>>3)
	}
}

func main() {
	Init()
	e := "qZlDH/USApHkRAu+fDvaf7PtbTqIhUbNirQzEZAGLYXq0MYE2Xwc7nE/WtioyQO0qnpEB5sIIRDF5vDDklo/mT2NjbxFgK7z6DO4HsAi3aOXw2FbndgFo07EdpZwBB9Ip/+v6B2Vr5hkjrKSiiaWLTF4QnHXUdbO+4M0JcC4bS8="
	v := V(e)
	key := getKey()
	fmt.Println(key.Decrypt(v))

}
