#!/usr/bin/env bash
# Provision Kibana data view + dashboard for DemoCare logs/stats.
#
# Idempotent: re-running overwrites the saved objects with the same IDs.
# Requires the elastic profile to be up (Elasticsearch + Kibana + Filebeat).
#
# Usage:
#   KIBANA_URL=http://127.0.0.1:5601 ./deploy/elastic/setup-kibana.sh
#
# Creates:
#   - data view  : knowledge-filebeat            (filebeat-*)
#   - visualizations:
#       knowledge-bot-qpm                        (queries/min)
#       knowledge-bot-latency                    (latency p50/p95 ms)
#       knowledge-bot-errors                     (failed queries/min)
#       knowledge-bot-top-senders                (top senders)
#       knowledge-logs-by-container              (log lines per container)
#       knowledge-ingest-batches                 (ingestor batch latency)
#       knowledge-errors-by-service              (errors per container)
#       knowledge-ollama-errors                  (Ollama/embed probe failures)
#       knowledge-ollama-activity                (Ollama/embed-related log volume)
#       knowledge-qdrant-container-logs          (Qdrant container throughput)
#       knowledge-qdrant-app-events              (upserts + bot Qdrant messages)
#   - dashboards:
#       knowledge-overview                       (DemoCare · Logs & Bot stats — общий)
#       knowledge-rag                            (RAG: запросы бота)
#       knowledge-ollama                         (Ollama/embed через логи bot/ingestor)
#       knowledge-qdrant                         (Qdrant: контейнер + приложение)

set -euo pipefail

KIBANA_URL="${KIBANA_URL:-http://127.0.0.1:5601}"
KIBANA_AUTH="${KIBANA_AUTH:-}" # e.g. "elastic:changeme" if security is enabled

curl_kbn() {
  local method="$1" path="$2" body="${3-}"
  local args=(-sS -X "$method" -H 'kbn-xsrf: true' -H 'Content-Type: application/json')
  if [[ -n "$KIBANA_AUTH" ]]; then args+=(-u "$KIBANA_AUTH"); fi
  if [[ -n "$body" ]]; then args+=(--data "$body"); fi
  curl "${args[@]}" "${KIBANA_URL}${path}"
}

wait_for_kibana() {
  echo "Waiting for Kibana at ${KIBANA_URL} ..."
  for i in $(seq 1 90); do
    local code
    code=$(curl -s -o /dev/null -w '%{http_code}' -H 'kbn-xsrf: true' "${KIBANA_URL}/api/status" 2>/dev/null || echo 000)
    if [[ "$code" == "200" ]]; then
      echo "Kibana responded (HTTP 200)."
      return 0
    fi
    sleep 2
  done
  echo "Kibana did not become available (last HTTP ${code})." >&2
  return 1
}

put_so() {
  # PUT /api/saved_objects/{type}/{id}?overwrite=true
  local type="$1" id="$2" body="$3"
  echo "  - $type/$id"
  curl_kbn POST "/api/saved_objects/${type}/${id}?overwrite=true" "$body" >/dev/null
}

ensure_data_view() {
  echo "Data view: knowledge-filebeat (filebeat-*)"
  # POST is fine even if it exists (we delete first to keep id stable).
  curl_kbn DELETE "/api/data_views/data_view/knowledge-filebeat" >/dev/null 2>&1 || true
  curl_kbn POST "/api/data_views/data_view" '{
    "data_view": {
      "id": "knowledge-filebeat",
      "title": "filebeat-*",
      "name": "DemoCare · filebeat",
      "timeFieldName": "@timestamp",
      "allowNoIndex": true
    },
    "override": true
  }' >/dev/null
}

# Helper to wrap a TSVB visState into a saved-object payload.
viz_payload() {
  # $1 = title, $2 = visState JSON (single line)
  local title="$1" vis="$2"
  jq -nc --arg title "$title" --arg vis "$vis" '{
    attributes: {
      title: $title,
      visState: $vis,
      uiStateJSON: "{}",
      description: "",
      version: 1,
      kibanaSavedObjectMeta: { searchSourceJSON: "{\"query\":{\"language\":\"kuery\",\"query\":\"\"},\"filter\":[]}" }
    },
    references: [
      { id: "knowledge-filebeat", name: "kibanaSavedObjectMeta.searchSourceJSON.index", type: "index-pattern" }
    ]
  }'
}

# --- TSVB visualizations ----------------------------------------------------

vis_bot_qpm='{
  "title":"Bot queries per minute","type":"metrics","aggs":[],
  "params":{
    "id":"v-bot-qpm","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"knowledge.event : \"bot_query\""},
    "series":[{
      "id":"s1","label":"Queries","color":"#54B399","chart_type":"line","fill":"0.5",
      "split_mode":"everything","stacked":"none","line_width":2,"point_size":1,"axis_position":"right",
      "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"m1","type":"count"}]
    }]
  }
}'

vis_bot_latency='{
  "title":"Bot latency (ms) p50/p95","type":"metrics","aggs":[],
  "params":{
    "id":"v-bot-lat","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"knowledge.event : \"bot_query\""},
    "series":[
      {
        "id":"s50","label":"p50","color":"#6092C0","chart_type":"line","fill":"0",
        "split_mode":"everything","stacked":"none","line_width":2,"point_size":1,"axis_position":"right",
        "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
        "metrics":[{"id":"m50","type":"percentile","field":"knowledge.latency_ms","percentiles":[{"id":"p50","mode":"line","value":"50","percentile":""}]}]
      },
      {
        "id":"s95","label":"p95","color":"#D36086","chart_type":"line","fill":"0",
        "split_mode":"everything","stacked":"none","line_width":2,"point_size":1,"axis_position":"right",
        "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
        "metrics":[{"id":"m95","type":"percentile","field":"knowledge.latency_ms","percentiles":[{"id":"p95","mode":"line","value":"95","percentile":""}]}]
      }
    ]
  }
}'

vis_bot_errors='{
  "title":"Bot failed queries per minute","type":"metrics","aggs":[],
  "params":{
    "id":"v-bot-err","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"knowledge.event : \"bot_query\" and knowledge.ok : false"},
    "series":[{
      "id":"se","label":"Failed","color":"#E7664C","chart_type":"bar","fill":"0.7",
      "split_mode":"everything","stacked":"none","line_width":1,"point_size":0,"axis_position":"right",
      "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"me","type":"count"}]
    }]
  }
}'

vis_top_senders='{
  "title":"Top bot users (by query count)","type":"metrics","aggs":[],
  "params":{
    "id":"v-top-sender","type":"top_n","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"knowledge.event : \"bot_query\""},
    "bar_color_rules":[{"id":"r1"}],
    "series":[{
      "id":"st","label":"queries","color":"#54B399","chart_type":"bar","fill":"0.7",
      "split_mode":"terms","stacked":"none","line_width":1,"point_size":0,"axis_position":"right",
      "terms_field":"knowledge.sender","terms_size":"10","terms_order_by":"_count","terms_direction":"desc",
      "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"mt","type":"count"}]
    }]
  }
}'

vis_logs_by_container='{
  "title":"Log lines by container","type":"metrics","aggs":[],
  "params":{
    "id":"v-logs-cn","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "series":[{
      "id":"sl","label":"{{container.labels.com_docker_compose_service}}","color":"#54B399",
      "chart_type":"bar","fill":"0.7","stacked":"stacked","line_width":1,"point_size":0,
      "axis_position":"right","formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "split_mode":"terms","terms_field":"container.labels.com_docker_compose_service",
      "terms_size":"10","terms_order_by":"_count","terms_direction":"desc",
      "metrics":[{"id":"ml","type":"count"}]
    }]
  }
}'

vis_errors_by_service='{
  "title":"Errors per service per minute","type":"metrics","aggs":[],
  "params":{
    "id":"v-err-svc","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"knowledge.level : \"ERROR\" or message : *panic*"},
    "series":[{
      "id":"sE","label":"{{container.labels.com_docker_compose_service}}","color":"#E7664C",
      "chart_type":"bar","fill":"0.7","stacked":"stacked","line_width":1,"point_size":0,
      "axis_position":"right","formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "split_mode":"terms","terms_field":"container.labels.com_docker_compose_service",
      "terms_size":"10","terms_order_by":"_count","terms_direction":"desc",
      "metrics":[{"id":"mE","type":"count"}]
    }]
  }
}'

vis_ingest_batches='{
  "title":"Ingestor batch latency (ms, avg)","type":"metrics","aggs":[],
  "params":{
    "id":"v-ing-batch","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"knowledge.event : \"ingest_batch_upserted\""},
    "series":[{
      "id":"sib","label":"avg ms","color":"#9170B8","chart_type":"line","fill":"0.3",
      "split_mode":"everything","stacked":"none","line_width":2,"point_size":1,
      "axis_position":"right","formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"mib","type":"avg","field":"knowledge.latency_ms"}]
    }]
  }
}'

# Ollama runs on the host; only bot/ingestor logs mention embed/Ollama errors.
vis_ollama_errors='{
  "title":"Ollama embed probe failures per minute","type":"metrics","aggs":[],
  "params":{
    "id":"v-ollama-err","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"container.labels.com_docker_compose_service : \"matrix-bot\" and knowledge.msg : \"ollama embed probe\" and knowledge.level : \"ERROR\""},
    "series":[{
      "id":"soe","label":"Failures","color":"#E7664C","chart_type":"bar","fill":"0.7",
      "split_mode":"everything","stacked":"none","line_width":1,"point_size":0,"axis_position":"right",
      "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"moe","type":"count"}]
    }]
  }
}'

vis_ollama_activity='{
  "title":"Ollama / embed related events per minute","type":"metrics","aggs":[],
  "params":{
    "id":"v-ollama-act","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"container.labels.com_docker_compose_service : (\"matrix-bot\" or \"kg-ingestor\") and (knowledge.msg : *embed* or knowledge.msg : *ollama* or knowledge.err : *ollama* or knowledge.err : *embeddings*)"},
    "series":[{
      "id":"soa","label":"Events","color":"#6092C0","chart_type":"line","fill":"0.35",
      "split_mode":"everything","stacked":"none","line_width":2,"point_size":1,"axis_position":"right",
      "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"moa","type":"count"}]
    }]
  }
}'

vis_qdrant_container='{
  "title":"Qdrant container log lines per minute","type":"metrics","aggs":[],
  "params":{
    "id":"v-qdr-svc","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"container.labels.com_docker_compose_service : \"qdrant\""},
    "series":[{
      "id":"sqc","label":"Lines","color":"#54B399","chart_type":"bar","fill":"0.5",
      "split_mode":"everything","stacked":"none","line_width":1,"point_size":0,"axis_position":"right",
      "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"mqc","type":"count"}]
    }]
  }
}'

vis_qdrant_app='{
  "title":"Qdrant app: upserts + bot messages per minute","type":"metrics","aggs":[],
  "params":{
    "id":"v-qdr-app","type":"timeseries","index_pattern":"filebeat-*","time_field":"@timestamp",
    "interval":"","axis_position":"left","axis_formatter":"number","show_grid":1,"show_legend":1,
    "default_index_pattern":"filebeat-*","default_timefield":"@timestamp","use_kibana_indexes":false,
    "filter":{"language":"kuery","query":"(container.labels.com_docker_compose_service : \"kg-ingestor\" and knowledge.event : \"ingest_batch_upserted\") or (container.labels.com_docker_compose_service : \"matrix-bot\" and knowledge.msg : *qdrant*)"},
    "series":[{
      "id":"sqa","label":"Events","color":"#9170B8","chart_type":"line","fill":"0.25",
      "split_mode":"everything","stacked":"none","line_width":2,"point_size":1,"axis_position":"right",
      "formatter":"number","separate_axis":0,"split_color_mode":"kibana",
      "metrics":[{"id":"mqa","type":"count"}]
    }]
  }
}'

create_visualizations() {
  echo "Visualizations:"
  put_so visualization knowledge-bot-qpm           "$(viz_payload 'DemoCare · Bot queries per minute' "$(echo "$vis_bot_qpm" | jq -c .)")"
  put_so visualization knowledge-bot-latency       "$(viz_payload 'DemoCare · Bot latency (ms)'         "$(echo "$vis_bot_latency" | jq -c .)")"
  put_so visualization knowledge-bot-errors        "$(viz_payload 'DemoCare · Bot failed queries'        "$(echo "$vis_bot_errors" | jq -c .)")"
  put_so visualization knowledge-bot-top-senders   "$(viz_payload 'DemoCare · Top bot users'             "$(echo "$vis_top_senders" | jq -c .)")"
  put_so visualization knowledge-logs-by-container "$(viz_payload 'DemoCare · Log lines by container'    "$(echo "$vis_logs_by_container" | jq -c .)")"
  put_so visualization knowledge-errors-by-service "$(viz_payload 'DemoCare · Errors per service'        "$(echo "$vis_errors_by_service" | jq -c .)")"
  put_so visualization knowledge-ingest-batches    "$(viz_payload 'DemoCare · Ingestor batch latency'    "$(echo "$vis_ingest_batches" | jq -c .)")"
  put_so visualization knowledge-ollama-errors     "$(viz_payload 'DemoCare · Ollama embed probe failures' "$(echo "$vis_ollama_errors" | jq -c .)")"
  put_so visualization knowledge-ollama-activity   "$(viz_payload 'DemoCare · Ollama/embed activity'      "$(echo "$vis_ollama_activity" | jq -c .)")"
  put_so visualization knowledge-qdrant-container-logs "$(viz_payload 'DemoCare · Qdrant container logs'  "$(echo "$vis_qdrant_container" | jq -c .)")"
  put_so visualization knowledge-qdrant-app-events "$(viz_payload 'DemoCare · Qdrant app events'        "$(echo "$vis_qdrant_app" | jq -c .)")"
}

# --- Dashboard --------------------------------------------------------------

create_dashboard() {
  echo "Dashboard: knowledge-overview"

  local panels
  panels=$(jq -cn '
    [
      { panelIndex:"1", gridData:{x:0,  y:0,  w:24, h:12, i:"1"}, version:"8.15.0", type:"visualization", panelRefName:"panel_1" },
      { panelIndex:"2", gridData:{x:24, y:0,  w:24, h:12, i:"2"}, version:"8.15.0", type:"visualization", panelRefName:"panel_2" },
      { panelIndex:"3", gridData:{x:0,  y:12, w:24, h:12, i:"3"}, version:"8.15.0", type:"visualization", panelRefName:"panel_3" },
      { panelIndex:"4", gridData:{x:24, y:12, w:24, h:12, i:"4"}, version:"8.15.0", type:"visualization", panelRefName:"panel_4" },
      { panelIndex:"5", gridData:{x:0,  y:24, w:24, h:12, i:"5"}, version:"8.15.0", type:"visualization", panelRefName:"panel_5" },
      { panelIndex:"6", gridData:{x:24, y:24, w:24, h:12, i:"6"}, version:"8.15.0", type:"visualization", panelRefName:"panel_6" },
      { panelIndex:"7", gridData:{x:0,  y:36, w:48, h:12, i:"7"}, version:"8.15.0", type:"visualization", panelRefName:"panel_7" }
    ]
  ')

  local refs
  refs=$(jq -cn '
    [
      { id:"knowledge-bot-qpm",           name:"panel_1", type:"visualization" },
      { id:"knowledge-bot-latency",       name:"panel_2", type:"visualization" },
      { id:"knowledge-bot-errors",        name:"panel_3", type:"visualization" },
      { id:"knowledge-bot-top-senders",   name:"panel_4", type:"visualization" },
      { id:"knowledge-logs-by-container", name:"panel_5", type:"visualization" },
      { id:"knowledge-errors-by-service", name:"panel_6", type:"visualization" },
      { id:"knowledge-ingest-batches",    name:"panel_7", type:"visualization" }
    ]
  ')

  local body
  body=$(jq -nc \
    --argjson panels "$panels" \
    --argjson refs "$refs" '
    {
      attributes: {
        title: "DemoCare · Logs & Bot stats",
        description: "Operational dashboard for the Matrix bot, ingestor and Compose containers (Filebeat → Elasticsearch).",
        hits: 0,
        panelsJSON: ($panels|tostring),
        optionsJSON: "{\"hidePanelTitles\":false,\"useMargins\":true,\"syncColors\":false,\"syncTooltips\":false,\"syncCursor\":true}",
        version: 1,
        timeRestore: true,
        timeFrom: "now-24h",
        timeTo: "now",
        refreshInterval: { display: "30 seconds", pause: false, value: 30000 },
        kibanaSavedObjectMeta: {
          searchSourceJSON: "{\"query\":{\"language\":\"kuery\",\"query\":\"\"},\"filter\":[]}"
        }
      },
      references: $refs
    }')

  curl_kbn POST "/api/saved_objects/dashboard/knowledge-overview?overwrite=true" "$body" >/dev/null
  echo "Dashboard: knowledge-overview — ${KIBANA_URL}/app/dashboards#/view/knowledge-overview"
}

# RAG = Matrix bot queries (same panels as overview subset).
create_dashboard_rag() {
  echo "Dashboard: knowledge-rag"
  local panels refs body
  panels=$(jq -cn '
    [
      { panelIndex:"1", gridData:{x:0,  y:0,  w:24, h:12, i:"1"}, version:"8.15.0", type:"visualization", panelRefName:"panel_1" },
      { panelIndex:"2", gridData:{x:24, y:0,  w:24, h:12, i:"2"}, version:"8.15.0", type:"visualization", panelRefName:"panel_2" },
      { panelIndex:"3", gridData:{x:0,  y:12, w:24, h:12, i:"3"}, version:"8.15.0", type:"visualization", panelRefName:"panel_3" },
      { panelIndex:"4", gridData:{x:24, y:12, w:24, h:12, i:"4"}, version:"8.15.0", type:"visualization", panelRefName:"panel_4" }
    ]
  ')
  refs=$(jq -cn '
    [
      { id:"knowledge-bot-qpm",         name:"panel_1", type:"visualization" },
      { id:"knowledge-bot-latency",     name:"panel_2", type:"visualization" },
      { id:"knowledge-bot-errors",      name:"panel_3", type:"visualization" },
      { id:"knowledge-bot-top-senders", name:"panel_4", type:"visualization" }
    ]
  ')
  body=$(jq -nc \
    --argjson panels "$panels" \
    --argjson refs "$refs" '
    {
      attributes: {
        title: "DemoCare · RAG (bot queries)",
        description: "Latency, volume, and failures for RAG answers (event=bot_query). Ollama/Qdrant/Neo4j must be reachable from the bot container.",
        hits: 0,
        panelsJSON: ($panels|tostring),
        optionsJSON: "{\"hidePanelTitles\":false,\"useMargins\":true,\"syncColors\":false,\"syncTooltips\":false,\"syncCursor\":true}",
        version: 1,
        timeRestore: true,
        timeFrom: "now-24h",
        timeTo: "now",
        refreshInterval: { display: "30 seconds", pause: false, value: 30000 },
        kibanaSavedObjectMeta: {
          searchSourceJSON: "{\"query\":{\"language\":\"kuery\",\"query\":\"\"},\"filter\":[]}"
        }
      },
      references: $refs
    }')
  curl_kbn POST "/api/saved_objects/dashboard/knowledge-rag?overwrite=true" "$body" >/dev/null
  echo "  ${KIBANA_URL}/app/dashboards#/view/knowledge-rag"
}

create_dashboard_ollama() {
  echo "Dashboard: knowledge-ollama"
  local panels refs body
  panels=$(jq -cn '
    [
      { panelIndex:"1", gridData:{x:0, y:0, w:48, h:14, i:"1"}, version:"8.15.0", type:"visualization", panelRefName:"panel_1" },
      { panelIndex:"2", gridData:{x:0, y:14, w:48, h:14, i:"2"}, version:"8.15.0", type:"visualization", panelRefName:"panel_2" }
    ]
  ')
  refs=$(jq -cn '
    [
      { id:"knowledge-ollama-errors",   name:"panel_1", type:"visualization" },
      { id:"knowledge-ollama-activity", name:"panel_2", type:"visualization" }
    ]
  ')
  body=$(jq -nc \
    --argjson panels "$panels" \
    --argjson refs "$refs" '
    {
      attributes: {
        title: "DemoCare · Ollama / embed (via logs)",
        description: "Ollama listens on the host; only errors and embed-related lines from matrix-bot / kg-ingestor appear here.",
        hits: 0,
        panelsJSON: ($panels|tostring),
        optionsJSON: "{\"hidePanelTitles\":false,\"useMargins\":true,\"syncColors\":false,\"syncTooltips\":false,\"syncCursor\":true}",
        version: 1,
        timeRestore: true,
        timeFrom: "now-24h",
        timeTo: "now",
        refreshInterval: { display: "30 seconds", pause: false, value: 30000 },
        kibanaSavedObjectMeta: {
          searchSourceJSON: "{\"query\":{\"language\":\"kuery\",\"query\":\"\"},\"filter\":[]}"
        }
      },
      references: $refs
    }')
  curl_kbn POST "/api/saved_objects/dashboard/knowledge-ollama?overwrite=true" "$body" >/dev/null
  echo "  ${KIBANA_URL}/app/dashboards#/view/knowledge-ollama"
}

create_dashboard_qdrant() {
  echo "Dashboard: knowledge-qdrant"
  local panels refs body
  panels=$(jq -cn '
    [
      { panelIndex:"1", gridData:{x:0, y:0, w:48, h:14, i:"1"}, version:"8.15.0", type:"visualization", panelRefName:"panel_1" },
      { panelIndex:"2", gridData:{x:0, y:14, w:48, h:14, i:"2"}, version:"8.15.0", type:"visualization", panelRefName:"panel_2" }
    ]
  ')
  refs=$(jq -cn '
    [
      { id:"knowledge-qdrant-container-logs", name:"panel_1", type:"visualization" },
      { id:"knowledge-qdrant-app-events",     name:"panel_2", type:"visualization" }
    ]
  ')
  body=$(jq -nc \
    --argjson panels "$panels" \
    --argjson refs "$refs" '
    {
      attributes: {
        title: "DemoCare · Qdrant",
        description: "Vector store: container log volume + ingest upserts and bot-side Qdrant messages.",
        hits: 0,
        panelsJSON: ($panels|tostring),
        optionsJSON: "{\"hidePanelTitles\":false,\"useMargins\":true,\"syncColors\":false,\"syncTooltips\":false,\"syncCursor\":true}",
        version: 1,
        timeRestore: true,
        timeFrom: "now-24h",
        timeTo: "now",
        refreshInterval: { display: "30 seconds", pause: false, value: 30000 },
        kibanaSavedObjectMeta: {
          searchSourceJSON: "{\"query\":{\"language\":\"kuery\",\"query\":\"\"},\"filter\":[]}"
        }
      },
      references: $refs
    }')
  curl_kbn POST "/api/saved_objects/dashboard/knowledge-qdrant?overwrite=true" "$body" >/dev/null
  echo "  ${KIBANA_URL}/app/dashboards#/view/knowledge-qdrant"
}

main() {
  command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }
  command -v curl >/dev/null || { echo "curl is required" >&2; exit 1; }
  wait_for_kibana
  ensure_data_view
  create_visualizations
  create_dashboard
  create_dashboard_rag
  create_dashboard_ollama
  create_dashboard_qdrant
  echo ""
  echo "All dashboards ready. Bot monitoring = knowledge-overview."
}

main "$@"
