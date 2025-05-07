package com.gregthemadmonk.mctg_serverbot_mod;

import org.slf4j.Logger;

import com.mojang.logging.LogUtils;

import net.minecraft.world.entity.player.Player;
import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.fml.ModContainer;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.common.NeoForge;
import net.neoforged.neoforge.event.ServerChatEvent;
import net.neoforged.neoforge.event.entity.living.LivingDeathEvent;

// The value here should match an entry in the META-INF/neoforge.mods.toml file
@Mod(value = MCTGMod.MODID, dist = Dist.DEDICATED_SERVER)
public class MCTGMod
{
    // Define mod id in a common place for everything to reference
    public static final String MODID = "mctg_serverbot_mod";
    // Directly reference a slf4j logger
    private static final Logger LOGGER = LogUtils.getLogger();

    public MCTGMod(IEventBus modEventBus, ModContainer modContainer)
    { NeoForge.EVENT_BUS.register(this); }

    @SubscribeEvent
    public void onPlayerMessage(ServerChatEvent event) {
        this.LOGGER.info(
            "CHAT"
            + event.getPlayer().getName().getString()
            + ""
            + event.getRawText()
        );
        event.setCanceled(true);
    } // <-- onPlayerMessage(event)

    @SubscribeEvent
    public void onPlayerDeath(LivingDeathEvent event) {
        if (event.getEntity() instanceof Player player) {
            this.LOGGER.info(
                "DEATH"
                + player.getName().getString()
                + ""
                + player.getCombatTracker().getDeathMessage().getString()
            );
        }
    } // <-- onPlayerDeath(event)
}
