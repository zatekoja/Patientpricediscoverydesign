-- enrich.lua - Lua script to enrich logs with service metadata based on container name

function enrich_logs(tag, timestamp, record)
    -- Extract container name from stream field
    local stream = record["stream"] or ""
    local container_name = ""

    -- Try to extract from com.docker.compose.service if available
    if record["attrs"] and record["attrs"]["com.docker.compose.service"] then
        container_name = record["attrs"]["com.docker.compose.service"]
    end

    -- Map container names to services
    if string.match(stream, "ppd_api") or container_name == "api" then
        record["service.name"] = "patient-price-discovery-api"
        record["service.type"] = "api"
        record["component"] = "backend"
        record["telemetry.sdk.language"] = "go"
    elseif string.match(stream, "ppd_graphql") or container_name == "graphql" then
        record["service.name"] = "patient-price-discovery-graphql"
        record["service.type"] = "graphql"
        record["component"] = "backend"
        record["telemetry.sdk.language"] = "go"
    elseif string.match(stream, "ppd_sse") or container_name == "sse" then
        record["service.name"] = "patient-price-discovery-sse"
        record["service.type"] = "sse"
        record["component"] = "backend"
        record["telemetry.sdk.language"] = "go"
    elseif string.match(stream, "ppd_provider") or container_name == "provider-api" then
        record["service.name"] = "patient-price-discovery-provider"
        record["service.type"] = "provider"
        record["component"] = "backend"
        record["telemetry.sdk.language"] = "nodejs"
    elseif string.match(stream, "ppd_postgres") or container_name == "postgres" then
        record["service.name"] = "postgres"
        record["db.type"] = "postgres"
        record["component"] = "database"
    elseif string.match(stream, "ppd_redis") or container_name == "redis" then
        record["service.name"] = "redis"
        record["db.type"] = "redis"
        record["component"] = "cache"
    elseif string.match(stream, "ppd_mongo") or container_name == "mongo" then
        record["service.name"] = "mongodb"
        record["db.type"] = "mongodb"
        record["component"] = "database"
    elseif string.match(stream, "ppd_typesense") or container_name == "typesense" then
        record["service.name"] = "typesense"
        record["component"] = "search"
    else
        record["service.name"] = "unknown"
    end

    return 2, timestamp, record
end

