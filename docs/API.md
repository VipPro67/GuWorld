# Backend API Map

Base path: `/api`.

Các route protected yêu cầu cookie/JWT hợp lệ qua `middleware.AuthRequired()` trừ khi ghi public.

## Public / auth / payment entrypoints

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/dictionary` | Public Gu + item dictionary. |
| GET | `/recipes` | Public refining recipes. |
| POST | `/auth/register` | Rate limited. |
| POST | `/auth/login` | Rate limited. |
| POST | `/auth/logout` | Public logout endpoint. |
| GET | `/auth/me` | Current player, protected. |
| GET | `/auth/google/login` | Google OAuth start, rate limited. |
| GET | `/auth/google/callback` | Google OAuth callback, rate limited. |
| GET | `/topup/packages` | Public topup packages. |
| POST | `/webhooks/payos` | PayOS webhook. |
| GET | `/admin/topup-orders` | Protected admin listing. |
| GET | `/admin/wallet-transactions` | Protected admin listing. |
| GET | `/wallet` | Protected wallet. |
| POST | `/topup/orders` | Protected create order, rate limited. |
| GET | `/topup/orders/:orderCode` | Protected order detail. |

## Player

| Method | Path |
| --- | --- |
| GET | `/player/events` |
| GET | `/player/status/:id` |
| GET | `/player/stats` |
| POST | `/player/explore/:id` |
| POST | `/player/use-item` |
| POST | `/player/recharge` |
| POST | `/player/clear-stress` |
| GET | `/player/profile/:username` |
| PUT | `/player/profile/settings` |
| GET | `/player/referrals` |
| POST | `/player/simulate-deposit` |
| POST | `/player/avatar` |
| POST | `/player/tick-regen` |

## Cultivation and quests

| Method | Path |
| --- | --- |
| POST | `/cultivate/break-wall` |
| POST | `/cultivate/use-relic-gu` |
| POST | `/cultivate/ascension` |
| POST | `/cultivate/tribulation` |
| POST | `/quest/complete` |

## Inventory, refining and killer moves

| Method | Path |
| --- | --- |
| POST | `/inventory/feed` |
| POST | `/inventory/feed-group` |
| POST | `/inventory/equip` |
| POST | `/inventory/status` |
| POST | `/inventory/set-vital` |
| POST | `/inventory/use-gu` |
| POST | `/inventory/use-support` |
| POST | `/inventory/drop` |
| POST | `/refine` |
| POST | `/refine/deduce` |
| POST | `/killer-moves/calculate` |
| POST | `/killer-moves` |
| GET | `/killer-moves` |
| GET | `/killer-moves/:id` |
| DELETE | `/killer-moves/:id` |
| POST | `/combat/cast-killer-move` |

## Exploration and combat

| Method | Path |
| --- | --- |
| POST | `/combat/attack` |
| POST | `/explore` |
| POST | `/explore/retreat/:id` |
| GET | `/explore/grid` |
| POST | `/explore/move` |

## Party, arena, mine

| Method | Path |
| --- | --- |
| POST | `/party/create` |
| GET | `/party/status` |
| POST | `/party/invite` |
| POST | `/party/leave` |
| GET | `/arena/leaderboard` |
| POST | `/arena/challenge` |
| GET | `/mine/slots` |
| POST | `/mine/occupy` |
| POST | `/mine/claim` |
| POST | `/mine/abandon` |
| POST | `/mine/attack` |

## Immortal aperture, dao marks and tribulations

| Method | Path |
| --- | --- |
| GET | `/immortal-aperture` |
| GET | `/immortal-aperture/nodes` |
| POST | `/immortal-aperture/nodes` |
| POST | `/immortal-aperture/nodes/:id/collect` |
| POST | `/immortal-aperture/nodes/:id/uninstall` |
| DELETE | `/immortal-aperture/nodes/:id` |
| POST | `/immortal-aperture/time-rate` |
| POST | `/immortal-aperture/labor` |
| POST | `/immortal-aperture/nodes/:id/repair` |
| GET | `/immortal-aperture/variant-humans` |
| GET | `/dao-marks` |
| GET | `/tribulations/active` |
| POST | `/tribulations/:eventId/attack` |
| POST | `/tribulations/:eventId/resolve` |

## Shop, marketplace and P2P trade

| Method | Path |
| --- | --- |
| GET | `/shop/packages` |
| GET | `/shop/list` |
| POST | `/shop/buy-package` |
| POST | `/shop/buy` |
| POST | `/shop/sell` |
| GET | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/listings` |
| POST | `/shop/bao-hoang/buy` |
| POST | `/shop/bao-hoang/bid` |
| POST | `/shop/bao-hoang/cancel` |
| POST | `/shop/bao-hoang/settle` |
| POST | `/trade/initiate` |
| GET | `/trade/active` |
| GET | `/trade/events` |
| POST | `/trade/accept/:id` |
| POST | `/trade/decline/:id` |
| POST | `/trade/cancel/:id` |
| POST | `/trade/offer/:id` |
| POST | `/trade/lock/:id` |
| POST | `/trade/confirm/:id` |

## Chat/mail, soul and instances

| Method | Path |
| --- | --- |
| POST | `/chat/mail/send` |
| GET | `/chat/mail/inbox` |
| POST | `/chat/bao-hoang/send` |
| GET | `/chat/bao-hoang/messages` |
| GET | `/chat/bao-hoang/events` |
| POST | `/soul/use-guts` |
| POST | `/soul/refine` |
| POST | `/soul/calm` |
| GET | `/instance/active` |
| POST | `/instance/enter` |
| POST | `/instance/action` |

## Sects

| Method | Path |
| --- | --- |
| GET | `/sects` |
| POST | `/sects/:id/join` |
| POST | `/sects/leave` |
| GET | `/sects/me` |
| POST | `/sects/promote` |
| POST | `/sects/donate/stones` |
| POST | `/sects/donate/materials` |
| POST | `/sects/donate/gu` |
| POST | `/sects/vault/buy` |
| GET | `/sects/missions` |
| POST | `/sects/missions/:missionId/accept` |
| POST | `/sects/missions/:missionId/complete` |
| POST | `/sects/create` |
| GET | `/sects/me/members` |
| POST | `/sects/members/:memberId/rank` |
| POST | `/sects/members/:memberId/kick` |
| POST | `/sects/missions/create` |
| POST | `/sects/vault/stock` |
| POST | `/sects/upgrade` |
| POST | `/sects/toggle-approval` |
| GET | `/sects/join-requests` |
| POST | `/sects/join-requests/:requestId/action` |
| GET | `/sects/promotion-requests` |
| POST | `/sects/promotion-requests/:requestId/action` |
| GET | `/sects/mines` |
| POST | `/sects/mines/claim` |
| POST | `/sects/mines/attack` |

## Feedback

| Method | Path | Ghi chú |
| --- | --- | --- |
| GET | `/feedback` | Public list/filter. |
| POST | `/feedback` | Protected submit. |
| POST | `/feedback/:id/vote` | Protected vote. |
| PUT | `/feedback/:id/status` | Protected status update. |
