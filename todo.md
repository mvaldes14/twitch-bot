# TODO
- Add messages for user events like follow
- Figure out messages for subs/etc when the time comes
- Add more commands to bot
- [x] branch for webhooks ✅ 2024-04-24
    - remove irc library
    - put secrets in env variables -> doppler
    - need callback on 443, done with kubernetes + cloudflared
    - different endpoints for different subscriptions - change callback
    - subscriptions to use:
        - channel.chat.message => for all incoming messages (scope: user:read:chat user:bot channel:bot)
        - channel.follow => new followers (scope: moderator:read:followers)
        - channel.subscribe => new subs (scope: channel:read:subscriptions)
    - check headers for type of events
        - header['Twitch-Eventsub-Message-Type'] == 'notification' to get event data
        - header['Twitch-Eventsub-Message-Type'] == 'webhook_callback_verification' to subscribe to events
    cmd -> http -> routes (types)
                -> helper (subscription)
                -> log/index (elastic)
- [x] send logs/activity to elastic ✅ 2024-04-28
    - [x] format all logs and statements into json for easy parsing ✅ 2024-04-28
    - [x] generate the credentials in elastic for ingestion ✅ 2024-04-28
    - [x] client working and indexing ✅ 2024-04-28
        -  [x] modify dns to point to kubernetes at deploy ✅ 2024-04-28
        - [ ] Generate makefile to run the process with .env
        - [ ] Add log events for handlers (chat, follow, sub)
- [ ] test the channel.channel_points_automatic_reward_redemption.add 
        scopes needed: 
            - channel:read:redemptions 
            - channel:manage:redemptions 
     send webhook or call to OBS
