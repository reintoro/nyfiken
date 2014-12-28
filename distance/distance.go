// Package distance measures the distances between two strings.
package distance

// NOTE: The ad-hoc function works well enough but it would be interesting to
// investigate other string distance functions. If it could consider "helol" and
// "hello" to be more similar than "hello" and "loleh" that would be great, thus
// differentiating between simple typos and distinct words. This could be a
// don't fix it if it ain't broken, but in the name of for fun and profit
// anything is fair game :)

// An ad-hoc function for a percentage difference between two strings.
func Approx(str1, str2 string) float64 {
	var sum1 float64
	for _, chr := range str1 {
		sum1 += float64(chr)
	}
	var sum2 float64
	for _, chr := range str2 {
		sum2 += float64(chr)
	}
	if sum1 > sum2 {
		return 100 - (float64(sum2/sum1) * 100)
	} else if sum2 > sum1 {
		return 100 - (float64(sum1/sum2) * 100)
	}
	return 0
}
