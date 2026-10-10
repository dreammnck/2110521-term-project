FROM node:22-alpine AS build
WORKDIR /repo
COPY package.json package-lock.json ./
COPY packages ./packages
COPY staff-web/package.json staff-web/
COPY resident-app/package.json resident-app/
RUN npm ci
ARG APP
COPY ${APP} ./${APP}
RUN npm run build -w ${APP}

FROM node:22-alpine
WORKDIR /srv
ARG APP
ENV NODE_ENV=production APP=${APP} HOSTNAME=0.0.0.0
COPY --from=build /repo/${APP}/.next/standalone ./
COPY --from=build /repo/${APP}/.next/static ./${APP}/.next/static
COPY --from=build /repo/${APP}/public ./${APP}/public
USER node
CMD node ${APP}/server.js
