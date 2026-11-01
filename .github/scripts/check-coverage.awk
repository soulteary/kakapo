# Check the exact statement-weighted total rather than a rounded display value.
NR == 1 {
    if ($0 !~ /^mode: (set|count|atomic)$/) invalid = 1
    next
}
NF != 3 || $2 !~ /^[0-9]+$/ || $3 !~ /^[0-9]+$/ {
    invalid = 1
    next
}
{
    total += $2
    if ($3 > 0) covered += $2
}
END {
    if (invalid || NR < 2 || total <= 0) {
        print "Invalid or empty coverage profile" > "/dev/stderr"
        exit 1
    }
    printf "Statement coverage: %d/%d (%.2f%%); required: 95%%\n", covered, total, covered * 100 / total
    if (covered * 100 < 95 * total) exit 1
}
