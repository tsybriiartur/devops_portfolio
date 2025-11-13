
#!/bin/bash
CPU_USAGE=$(top -bn1 | grep "Cpu(s)" | \
sed "s/.*, *\([0-9.]*\)%* id.*/\1/" | \
awk '{print 100 - $1}') 
echo "CPU: $CPU_USAGE"
RAM_USAGE=$(free | grep Mem | \
awk '{printf("%.0f", ($3/$2) * 100.0)}')
echo "RAM: $RAM_USAGE"
df -h | awk '$6 == "/" {
 gsub(/%/, "", $5)
 print $6 ": " $5 "%"
}'

