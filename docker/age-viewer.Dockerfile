FROM node:16-alpine

# Install git and pm2
RUN apk add --no-cache git && npm install -g pm2

WORKDIR /src

# Clone the Apache AGE Viewer repository
RUN git clone --depth 1 https://github.com/apache/age-viewer.git .

# Install root dependencies
RUN npm install

# Install frontend dependencies
WORKDIR /src/frontend
RUN npm install

# Install backend dependencies with additional babel runtime
WORKDIR /src/backend
RUN npm install && npm install @babel/runtime

WORKDIR /src

# Add PostgreSQL 17 compatibility (age-viewer only ships meta_data.sql for PG 11-15)
RUN cp -r /src/backend/sql/15 /src/backend/sql/17

EXPOSE 3000

CMD ["npm", "run", "start"]
