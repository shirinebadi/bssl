package sike

// Helpers

// uint128 representation
type uint128 struct {
	H, L uint64
}

func addc64(cin, a, b uint64) (ret, cout uint64) {
	ret = cin
	ret = ret + a
	if ret < a {
		cout = 1
	}
	ret = ret + b
	if ret < b {
		cout = 1
	}

	return
}

func subc64(bIn, a, b uint64) (ret, bOut uint64) {
	tmp := a - bIn
	if tmp > a {
		bOut = 1
	}
	ret = tmp - b
	if ret > tmp {
		bOut = 1
	}
	return
}

func mul64(a, b uint64) (res uint128) {
	var al, bl, ah, bh, albl, albh, ahbl, ahbh uint64
	var res1, res2, res3 uint64
	var carry, maskL, maskH, temp uint64

	maskL = (^maskL) >> 32
	maskH = ^maskL

	al = a & maskL
	ah = a >> 32
	bl = b & maskL
	bh = b >> 32

	albl = al * bl
	albh = al * bh
	ahbl = ah * bl
	ahbh = ah * bh
	res.L = albl & maskL

	res1 = albl >> 32
	res2 = ahbl & maskL
	res3 = albh & maskL
	temp = res1 + res2 + res3
	carry = temp >> 32
	res.L ^= temp << 32

	res1 = ahbl >> 32
	res2 = albh >> 32
	res3 = ahbh & maskL
	temp = res1 + res2 + res3 + carry
	res.H = temp & maskL
	carry = temp & maskH
	res.H ^= (ahbh & maskH) + carry
	return
}

// Fp implementation

// Compute z = x + y (mod 2*p).
func fpAddRdc(z, x, y *Fp) {
	var carry uint64

	// z=x+y % p503
	for i := 0; i < FP_WORDS; i++ {
		z[i], carry = addc64(carry, x[i], y[i])
	}

	// z = z - p503x2
	carry = 0
	for i := 0; i < FP_WORDS; i++ {
		z[i], carry = subc64(carry, z[i], p503x2[i])
	}

	// if z<0 add p503x2 back
	mask := uint64(0 - carry)
	carry = 0
	for i := 0; i < FP_WORDS; i++ {
		z[i], carry = addc64(carry, z[i], p503x2[i]&mask)
	}
}

// Compute z = x - y (mod 2*p).
func fpSubRdc(z, x, y *Fp) {
	var borrow uint64

	// z = z - p503x2
	for i := 0; i < FP_WORDS; i++ {
		z[i], borrow = subc64(borrow, x[i], y[i])
	}

	// if z<0 add p503x2 back
	mask := uint64(0 - borrow)
	borrow = 0
	for i := 0; i < FP_WORDS; i++ {
		z[i], borrow = addc64(borrow, z[i], p503x2[i]&mask)
	}
}

// Reduce a field element in [0, 2*p) to one in [0,p).
func fpRdcP(x *Fp) {
	var borrow, mask uint64
	for i := 0; i < FP_WORDS; i++ {
		x[i], borrow = subc64(borrow, x[i], p503[i])
	}

	// Sets all bits if borrow = 1
	mask = 0 - borrow
	borrow = 0
	for i := 0; i < FP_WORDS; i++ {
		x[i], borrow = addc64(borrow, x[i], p503[i]&mask)
	}
}

// Implementation doesn't actually depend on a prime field.
func fpSwapCond(x, y *Fp, mask uint8) {
	if mask != 0 {
		var tmp Fp
		copy(tmp[:], y[:])
		copy(y[:], x[:])
		copy(x[:], tmp[:])
	}
}

// Compute z = x * y.
func fpMul(z *FpX2, x, y *Fp) {
	var u, v, t uint64
	var carry uint64
	var uv uint128

	for i := uint64(0); i < FP_WORDS; i++ {
		for j := uint64(0); j <= i; j++ {
			uv = mul64(x[j], y[i-j])
			v, carry = addc64(0, uv.L, v)
			u, carry = addc64(carry, uv.H, u)
			t += carry
		}
		z[i] = v
		v = u
		u = t
		t = 0
	}

	for i := FP_WORDS; i < (2*FP_WORDS)-1; i++ {
		for j := i - FP_WORDS + 1; j < FP_WORDS; j++ {
			uv = mul64(x[j], y[i-j])
			v, carry = addc64(0, uv.L, v)
			u, carry = addc64(carry, uv.H, u)
			t += carry
		}
		z[i] = v
		v = u
		u = t
		t = 0
	}
	z[2*FP_WORDS-1] = v
}

// Perform Montgomery reduction: set z = x R^{-1} (mod 2*p)
// with R=2^512. Destroys the input value.
func fpMontRdc(z *Fp, x *FpX2) {
	var carry, t, u, v uint64
	var uv uint128
	var count int

	count = 3 // number of 0 digits in the least significat part of p503 + 1

	for i := 0; i < FP_WORDS; i++ {
		for j := 0; j < i; j++ {
			if j < (i - count + 1) {
				uv = mul64(z[j], p503p1[i-j])
				v, carry = addc64(0, uv.L, v)
				u, carry = addc64(carry, uv.H, u)
				t += carry
			}
		}
		v, carry = addc64(0, v, x[i])
		u, carry = addc64(carry, u, 0)
		t += carry

		z[i] = v
		v = u
		u = t
		t = 0
	}

	for i := FP_WORDS; i < 2*FP_WORDS-1; i++ {
		if count > 0 {
			count--
		}
		for j := i - FP_WORDS + 1; j < FP_WORDS; j++ {
			if j < (FP_WORDS - count) {
				uv = mul64(z[j], p503p1[i-j])
				v, carry = addc64(0, uv.L, v)
				u, carry = addc64(carry, uv.H, u)
				t += carry
			}
		}
		v, carry = addc64(0, v, x[i])
		u, carry = addc64(carry, u, 0)

		t += carry
		z[i-FP_WORDS] = v
		v = u
		u = t
		t = 0
	}
	v, carry = addc64(0, v, x[2*FP_WORDS-1])
	z[FP_WORDS-1] = v
}

// Compute z = x + y, without reducing mod p.
func fp2Add(z, x, y *FpX2) {
	var carry uint64
	for i := 0; i < 2*FP_WORDS; i++ {
		z[i], carry = addc64(carry, x[i], y[i])
	}
}

// Compute z = x - y, without reducing mod p.
func fp2Sub(z, x, y *FpX2) {
	var borrow, mask uint64
	for i := 0; i < 2*FP_WORDS; i++ {
		z[i], borrow = subc64(borrow, x[i], y[i])
	}

	// Sets all bits if borrow = 1
	mask = 0 - borrow
	borrow = 0
	for i := FP_WORDS; i < 2*FP_WORDS; i++ {
		z[i], borrow = addc64(borrow, z[i], p503[i-FP_WORDS]&mask)
	}
}

// Montgomery multiplication. Input values must be already
// in Montgomery domain.
func fpMulRdc(dest, lhs, rhs *Fp) {
	a := lhs // = a*R
	b := rhs // = b*R

	var ab FpX2
	fpMul(&ab, a, b)     // = a*b*R*R
	fpMontRdc(dest, &ab) // = a*b*R mod p
}

// Set dest = x^((p-3)/4).  If x is square, this is 1/sqrt(x).
// Uses variation of sliding-window algorithm from with window size
// of 5 and least to most significant bit sliding (left-to-right)
// See HAC 14.85 for general description.
//
// Allowed to overlap x with dest.
// All values in Montgomery domains
func p34(dest, x *Fp) {

	// Set dest = x^(2^k), for k >= 1, by repeated squarings.
	pow2k := func(dest, x *Fp, k uint8) {
		fpMulRdc(dest, x, x)
		for i := uint8(1); i < k; i++ {
			fpMulRdc(dest, dest, dest)
		}
	}
	// Sliding-window strategy computed with etc/scripts/sliding_window_strat_calc.py
	//
	// This performs sum(powStrategy) + 1 squarings and len(lookup) + len(mulStrategy)
	// multiplications.
	powStrategy := []uint8{1, 12, 5, 5, 2, 7, 11, 3, 8, 4, 11, 4, 7, 5, 6, 3, 7, 5, 7, 2, 12, 5, 6, 4, 6, 8, 6, 4, 7, 5, 5, 8, 5, 8, 5, 5, 8, 9, 3, 6, 2, 10, 6, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 3}
	mulStrategy := []uint8{0, 12, 11, 10, 0, 1, 8, 3, 7, 1, 8, 3, 6, 7, 14, 2, 14, 14, 9, 0, 13, 9, 15, 5, 12, 7, 13, 7, 15, 6, 7, 9, 0, 5, 7, 6, 8, 8, 3, 7, 0, 10, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 3}

	// Precompute lookup table of odd multiples of x for window
	// size k=5.
	lookup := [16]Fp{}
	var xx Fp
	fpMulRdc(&xx, x, x)
	lookup[0] = *x
	for i := 1; i < 16; i++ {
		fpMulRdc(&lookup[i], &lookup[i-1], &xx)
	}

	// Now lookup = {x, x^3, x^5, ... }
	// so that lookup[i] = x^{2*i + 1}
	// so that lookup[k/2] = x^k, for odd k
	*dest = lookup[mulStrategy[0]]
	for i := uint8(1); i < uint8(len(powStrategy)); i++ {
		pow2k(dest, dest, powStrategy[i])
		fpMulRdc(dest, dest, &lookup[mulStrategy[i]])
	}
}

func add(dest, lhs, rhs *Fp2) {
	fpAddRdc(&dest.A, &lhs.A, &rhs.A)
	fpAddRdc(&dest.B, &lhs.B, &rhs.B)
}

func sub(dest, lhs, rhs *Fp2) {
	fpSubRdc(&dest.A, &lhs.A, &rhs.A)
	fpSubRdc(&dest.B, &lhs.B, &rhs.B)
}

func mul(dest, lhs, rhs *Fp2) {
	// Let (a,b,c,d) = (lhs.a,lhs.b,rhs.a,rhs.b).
	a := &lhs.A
	b := &lhs.B
	c := &rhs.A
	d := &rhs.B

	// We want to compute
	//
	// (a + bi)*(c + di) = (a*c - b*d) + (a*d + b*c)i
	//
	// Use Karatsuba's trick: note that
	//
	// (b - a)*(c - d) = (b*c + a*d) - a*c - b*d
	//
	// so (a*d + b*c) = (b-a)*(c-d) + a*c + b*d.

	var ac, bd FpX2
	fpMul(&ac, a, c) // = a*c*R*R
	fpMul(&bd, b, d) // = b*d*R*R

	var b_minus_a, c_minus_d Fp
	fpSubRdc(&b_minus_a, b, a) // = (b-a)*R
	fpSubRdc(&c_minus_d, c, d) // = (c-d)*R

	var ad_plus_bc FpX2
	fpMul(&ad_plus_bc, &b_minus_a, &c_minus_d) // = (b-a)*(c-d)*R*R
	fp2Add(&ad_plus_bc, &ad_plus_bc, &ac)      // = ((b-a)*(c-d) + a*c)*R*R
	fp2Add(&ad_plus_bc, &ad_plus_bc, &bd)      // = ((b-a)*(c-d) + a*c + b*d)*R*R

	fpMontRdc(&dest.B, &ad_plus_bc) // = (a*d + b*c)*R mod p

	var ac_minus_bd FpX2
	fp2Sub(&ac_minus_bd, &ac, &bd)   // = (a*c - b*d)*R*R
	fpMontRdc(&dest.A, &ac_minus_bd) // = (a*c - b*d)*R mod p
}

func inv(dest, x *Fp2) {
	var a2PlusB2 Fp
	var asq, bsq FpX2
	var ac FpX2
	var minusB Fp
	var minusBC FpX2

	a := &x.A
	b := &x.B

	// We want to compute
	//
	//    1          1     (a - bi)	    (a - bi)
	// -------- = -------- -------- = -----------
	// (a + bi)   (a + bi) (a - bi)   (a^2 + b^2)
	//
	// Letting c = 1/(a^2 + b^2), this is
	//
	// 1/(a+bi) = a*c - b*ci.

	fpMul(&asq, a, a)          // = a*a*R*R
	fpMul(&bsq, b, b)          // = b*b*R*R
	fp2Add(&asq, &asq, &bsq)   // = (a^2 + b^2)*R*R
	fpMontRdc(&a2PlusB2, &asq) // = (a^2 + b^2)*R mod p
	// Now a2PlusB2 = a^2 + b^2

	inv := a2PlusB2
	fpMulRdc(&inv, &a2PlusB2, &a2PlusB2)
	p34(&inv, &inv)
	fpMulRdc(&inv, &inv, &inv)
	fpMulRdc(&inv, &inv, &a2PlusB2)

	fpMul(&ac, a, &inv)
	fpMontRdc(&dest.A, &ac)

	fpSubRdc(&minusB, &minusB, b)
	fpMul(&minusBC, &minusB, &inv)
	fpMontRdc(&dest.B, &minusBC)
}

func sqr(dest, x *Fp2) {
	var a2, aPlusB, aMinusB Fp
	var a2MinB2, ab2 FpX2

	a := &x.A
	b := &x.B

	// (a + bi)*(a + bi) = (a^2 - b^2) + 2abi.
	fpAddRdc(&a2, a, a)                // = a*R + a*R = 2*a*R
	fpAddRdc(&aPlusB, a, b)            // = a*R + b*R = (a+b)*R
	fpSubRdc(&aMinusB, a, b)           // = a*R - b*R = (a-b)*R
	fpMul(&a2MinB2, &aPlusB, &aMinusB) // = (a+b)*(a-b)*R*R = (a^2 - b^2)*R*R
	fpMul(&ab2, &a2, b)                // = 2*a*b*R*R
	fpMontRdc(&dest.A, &a2MinB2)       // = (a^2 - b^2)*R mod p
	fpMontRdc(&dest.B, &ab2)           // = 2*a*b*R mod p
}

// In case choice == 1, performs following swap in constant time:
// 	xPx <-> xQx
//	xPz <-> xQz
// Otherwise returns xPx, xPz, xQx, xQz unchanged
func condSwap(xPx, xPz, xQx, xQz *Fp2, choice uint8) {
	fpSwapCond(&xPx.A, &xQx.A, choice)
	fpSwapCond(&xPx.B, &xQx.B, choice)
	fpSwapCond(&xPz.A, &xQz.A, choice)
	fpSwapCond(&xPz.B, &xQz.B, choice)
}
