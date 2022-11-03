local gm = require("gm.init")
local json = require("json")

function GMStub:on_init()
end

function GMStub:on_destroy()
end

function GMStub:get_gm_list()
    -- 复制一份, 移除无关信息
    local cp = table.deep_copy(gm:get_gm_list())
    for _, gmTable in pairs(cp) do
        for k, v in pairs(gmTable) do
            if k == "value" then
                for _, gm in pairs(v) do
                    gm.callback = nil
                end
            end
        end
    end
    return json.encode(cp)
end