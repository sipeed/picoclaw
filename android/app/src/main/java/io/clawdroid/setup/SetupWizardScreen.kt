package io.clawdroid.setup

import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedContent
import androidx.compose.animation.slideInHorizontally
import androidx.compose.animation.slideOutHorizontally
import androidx.compose.animation.togetherWith
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.drawBehind
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.Color
import io.clawdroid.core.ui.theme.DeepBlack
import io.clawdroid.core.ui.theme.GradientCyan
import io.clawdroid.core.ui.theme.GradientPurple
import org.koin.compose.viewmodel.koinViewModel

@Composable
fun SetupWizardScreen(
    onSetupComplete: () -> Unit,
    viewModel: SetupViewModel = koinViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()

    // Step 0 (Gateway/auth): suppress back entirely
    // Step 1+: go to previous step
    BackHandler(enabled = true) {
        if (uiState.currentStep > 0) {
            viewModel.previousStep()
        }
        // Step 0: do nothing (suppress back)
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(DeepBlack)
            .drawBehind {
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(
                            GradientCyan.copy(alpha = 0.07f),
                            Color.Transparent,
                        ),
                        center = Offset(size.width * 0.15f, size.height * 0.1f),
                        radius = size.width * 0.8f,
                    ),
                )
                drawCircle(
                    brush = Brush.radialGradient(
                        colors = listOf(
                            GradientPurple.copy(alpha = 0.07f),
                            Color.Transparent,
                        ),
                        center = Offset(size.width * 0.85f, size.height * 0.9f),
                        radius = size.width * 0.7f,
                    ),
                )
            },
    ) {
        AnimatedContent(
            targetState = uiState.currentStep,
            transitionSpec = {
                if (targetState > initialState) {
                    slideInHorizontally { it } togetherWith slideOutHorizontally { -it }
                } else {
                    slideInHorizontally { -it } togetherWith slideOutHorizontally { it }
                }
            },
            label = "setup_step",
        ) { step ->
            when (step) {
                0 -> SetupStep1GatewayScreen(viewModel)
                1 -> SetupStep2LlmScreen(viewModel)
                2 -> SetupStep3WorkspaceScreen(viewModel)
                3 -> SetupCompleteScreen(viewModel, onSetupComplete)
            }
        }
    }
}
