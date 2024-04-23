# TODO
- Add messages for user events like follow
- Figure out messages for subs/etc when the time comes
- Add more commands to bot
- branch for webhooks
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
