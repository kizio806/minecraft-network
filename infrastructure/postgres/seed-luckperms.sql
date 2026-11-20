-- ==============================================================================
-- LuckPerms 5-Rank Hierarchy Schema & Seed for PostgreSQL
-- ==============================================================================

CREATE TABLE IF NOT EXISTS luckperms_players (
    unique_id VARCHAR(36) NOT NULL PRIMARY KEY,
    username VARCHAR(16) NOT NULL,
    primary_group VARCHAR(36) NOT NULL
);
CREATE INDEX IF NOT EXISTS luckperms_players_username_idx ON luckperms_players (username);

CREATE TABLE IF NOT EXISTS luckperms_user_permissions (
    id SERIAL PRIMARY KEY,
    unique_id VARCHAR(36) NOT NULL,
    permission VARCHAR(200) NOT NULL,
    value BOOLEAN NOT NULL,
    server VARCHAR(36) NOT NULL DEFAULT 'global',
    world VARCHAR(64) NOT NULL DEFAULT 'global',
    expiry BIGINT NOT NULL DEFAULT 0,
    contexts VARCHAR(200) NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS luckperms_user_permissions_uid_idx ON luckperms_user_permissions (unique_id);

CREATE TABLE IF NOT EXISTS luckperms_actions (
    id SERIAL PRIMARY KEY,
    time BIGINT NOT NULL,
    actor_uuid VARCHAR(36) NOT NULL,
    actor_name VARCHAR(100) NOT NULL,
    type CHAR(1) NOT NULL,
    acted_uuid VARCHAR(36) NOT NULL,
    acted_name VARCHAR(36) NOT NULL,
    action VARCHAR(300) NOT NULL
);

CREATE TABLE IF NOT EXISTS luckperms_tracks (
    name VARCHAR(36) NOT NULL PRIMARY KEY,
    groups TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS luckperms_groups (
    name VARCHAR(36) NOT NULL PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS luckperms_group_permissions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(36) NOT NULL REFERENCES luckperms_groups(name) ON DELETE CASCADE,
    permission VARCHAR(200) NOT NULL,
    value BOOLEAN NOT NULL,
    server VARCHAR(36) NOT NULL DEFAULT 'global',
    world VARCHAR(64) NOT NULL DEFAULT 'global',
    expiry BIGINT NOT NULL DEFAULT 0,
    contexts VARCHAR(200) NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS luckperms_group_permissions_name_idx ON luckperms_group_permissions (name);

-- ------------------------------------------------------------------------------
-- 1. Insert Groups (5-Tier Model)
-- ------------------------------------------------------------------------------
INSERT INTO luckperms_groups (name) VALUES
    ('default'),
    ('helper'),
    ('moderator'),
    ('administrator'),
    ('owner')
ON CONFLICT (name) DO NOTHING;

-- ------------------------------------------------------------------------------
-- 2. Insert Tracks
-- ------------------------------------------------------------------------------
INSERT INTO luckperms_tracks (name, groups) VALUES
    ('staff', '["helper", "moderator", "administrator", "owner"]')
ON CONFLICT (name) DO UPDATE SET groups = EXCLUDED.groups;

-- ------------------------------------------------------------------------------
-- 3. Group Weights & Prefixes
-- ------------------------------------------------------------------------------
INSERT INTO luckperms_group_permissions (name, permission, value, server, world, expiry, contexts) VALUES
    -- default
    ('default', 'weight.10', true, 'global', 'global', 0, '{}'),
    ('default', 'prefix.10.&7Player &8» &f', true, 'global', 'global', 0, '{}'),
    -- helper
    ('helper', 'weight.40', true, 'global', 'global', 0, '{}'),
    ('helper', 'prefix.40.&#2ecc71&lHELPER &#2ecc71» &f', true, 'global', 'global', 0, '{}'),
    ('helper', 'group.default', true, 'global', 'global', 0, '{}'),
    -- moderator
    ('moderator', 'weight.60', true, 'global', 'global', 0, '{}'),
    ('moderator', 'prefix.60.&#00b4db&lMOD &#00b4db» &f', true, 'global', 'global', 0, '{}'),
    ('moderator', 'group.helper', true, 'global', 'global', 0, '{}'),
    -- administrator
    ('administrator', 'weight.80', true, 'global', 'global', 0, '{}'),
    ('administrator', 'prefix.80.&#e65c00&lADMIN &#e65c00» &f', true, 'global', 'global', 0, '{}'),
    ('administrator', 'group.moderator', true, 'global', 'global', 0, '{}'),
    -- owner
    ('owner', 'weight.100', true, 'global', 'global', 0, '{}'),
    ('owner', 'prefix.100.&#ff4e50&lOWNER &#ff4e50» &f', true, 'global', 'global', 0, '{}'),
    ('owner', 'group.administrator', true, 'global', 'global', 0, '{}')
ON CONFLICT DO NOTHING;

-- ------------------------------------------------------------------------------
-- 4. Functional Permissions
-- ------------------------------------------------------------------------------
INSERT INTO luckperms_group_permissions (name, permission, value, server, world, expiry, contexts) VALUES
    -- default permissions
    ('default', 'deluxemenus.menu', true, 'global', 'global', 0, '{}'),
    ('default', 'deluxemenus.menu.navigation', true, 'global', 'global', 0, '{}'),
    ('default', 'deluxemenus.open', true, 'global', 'global', 0, '{}'),
    ('default', 'itemjoin.use', true, 'global', 'global', 0, '{}'),
    ('default', 'parkour.basic', true, 'global', 'global', 0, '{}'),
    ('default', 'essentials.spawn', true, 'global', 'global', 0, '{}'),
    ('default', 'essentials.msg', true, 'global', 'global', 0, '{}'),
    ('default', 'essentials.home', true, 'global', 'global', 0, '{}'),
    ('default', 'essentials.sethome', true, 'global', 'global', 0, '{}'),
    ('default', 'essentials.tpa', true, 'global', 'global', 0, '{}'),
    ('default', 'essentials.tpaccept', true, 'global', 'global', 0, '{}'),
    ('default', 'essentials.tpdeny', true, 'global', 'global', 0, '{}'),
    ('default', 'pvpmanager.pvp', true, 'global', 'global', 0, '{}'),
    ('default', 'ultimateteams.user', true, 'global', 'global', 0, '{}'),
    ('default', 'gsit.sit', true, 'global', 'global', 0, '{}'),
    ('default', 'gsit.lay', true, 'global', 'global', 0, '{}'),
    ('default', 'gsit.crawl', true, 'global', 'global', 0, '{}'),
    ('default', 'gsit.spin', true, 'global', 'global', 0, '{}'),
    ('default', 'bettersleeping.sleep', true, 'global', 'global', 0, '{}'),
    ('default', 'ecoskills.skills', true, 'global', 'global', 0, '{}'),
    ('default', 'interactivechat.module.item', true, 'global', 'global', 0, '{}'),
    ('default', 'interactivechat.module.inv', true, 'global', 'global', 0, '{}'),
    ('default', 'interactivechat.module.ender', true, 'global', 'global', 0, '{}'),
    ('default', 'voicechat.listen', true, 'global', 'global', 0, '{}'),
    ('default', 'voicechat.speak', true, 'global', 'global', 0, '{}'),
    -- security blocks for default
    ('default', 'bukkit.command.plugins', false, 'global', 'global', 0, '{}'),
    ('default', 'bukkit.command.version', false, 'global', 'global', 0, '{}'),
    ('default', 'bukkit.command.help', false, 'global', 'global', 0, '{}'),
    ('default', 'minecraft.command.me', false, 'global', 'global', 0, '{}'),
    -- helper permissions
    ('helper', 'libertybans.mute', true, 'global', 'global', 0, '{}'),
    ('helper', 'libertybans.warn', true, 'global', 'global', 0, '{}'),
    ('helper', 'libertybans.kick', true, 'global', 'global', 0, '{}'),
    ('helper', 'libertybans.history', true, 'global', 'global', 0, '{}'),
    ('helper', 'essentials.socialspy', true, 'global', 'global', 0, '{}'),
    ('helper', 'essentials.helpop.receive', true, 'global', 'global', 0, '{}'),
    -- moderator permissions
    ('moderator', 'libertybans.ban', true, 'global', 'global', 0, '{}'),
    ('moderator', 'libertybans.tempban', true, 'global', 'global', 0, '{}'),
    ('moderator', 'libertybans.unban', true, 'global', 'global', 0, '{}'),
    ('moderator', 'libertybans.unmute', true, 'global', 'global', 0, '{}'),
    ('moderator', 'coreprotect.inspect', true, 'global', 'global', 0, '{}'),
    ('moderator', 'coreprotect.lookup', true, 'global', 'global', 0, '{}'),
    ('moderator', 'coreprotect.rollback', true, 'global', 'global', 0, '{}'),
    ('moderator', 'invseeplusplus.invsee', true, 'global', 'global', 0, '{}'),
    ('moderator', 'invseeplusplus.endersee', true, 'global', 'global', 0, '{}'),
    ('moderator', 'essentials.tp', true, 'global', 'global', 0, '{}'),
    ('moderator', 'essentials.tphere', true, 'global', 'global', 0, '{}'),
    ('moderator', 'essentials.vanish', true, 'global', 'global', 0, '{}'),
    ('moderator', 'grim.alerts', true, 'global', 'global', 0, '{}'),
    -- administrator permissions
    ('administrator', 'inventoryrollbackplus.admin', true, 'global', 'global', 0, '{}'),
    ('administrator', 'coreprotect.*', true, 'global', 'global', 0, '{}'),
    ('administrator', 'libertybans.*', true, 'global', 'global', 0, '{}'),
    ('administrator', 'nros.anticheat.admin', true, 'global', 'global', 0, '{}'),
    ('administrator', 'worldedit.*', true, 'global', 'global', 0, '{}'),
    ('administrator', 'worldguard.*', true, 'global', 'global', 0, '{}'),
    ('administrator', 'chunky.chunky', true, 'global', 'global', 0, '{}'),
    ('administrator', 'chunkyborder.admin', true, 'global', 'global', 0, '{}'),
    ('administrator', 'parkour.admin', true, 'global', 'global', 0, '{}'),
    ('administrator', 'spark.*', true, 'global', 'global', 0, '{}'),
    ('administrator', 'essentials.*', true, 'global', 'global', 0, '{}'),
    ('administrator', 'luckperms.*', true, 'global', 'global', 0, '{}'),
    -- owner permissions
    ('owner', '*', true, 'global', 'global', 0, '{}')
ON CONFLICT DO NOTHING;
