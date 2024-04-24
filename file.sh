id=(
"c75766bb-fc4c-4e32-a252-71227a2fe669"
)

for i in "${id[@]}"; do
  curl -X DELETE "https://api.twitch.tv/helix/eventsub/subscriptions?id=$i" -H 'Authorization: Bearer qj9bde9tnkjj5p54f39k3zpwsazrs7' -H 'Client-Id: wgcysot4d4snuxsk8g5vn3kjd5v0fv'
done


