module Farmation.Portal.Is exposing (InjectorSentry, deviceToInjectorSentry)

import Data.Device
import Data.Sample exposing (Sample)
import Element exposing (..)


type alias InjectorSentry =
    { inputWaterOn : Bool
    , inputIrrigator : Bool
    , inputInjector : Bool
    , armed : Bool
    , outputShutdown : Bool
    , outputInjector : Bool
    , flowRate : Float
    , currentTankVolume : Float
    , tankCapacity : Float
    , pressureMin : Float
    , pressureMax : Float
    , flowRateTarget : Float
    , flowWindowLow : Float
    , flowWindowHigh : Float
    }


deviceToInjectorSentry : Data.Device.Device -> InjectorSentry
deviceToInjectorSentry device =
    let
        is =
            InjectorSentry False False False False False False 0 0 0 0 0 0 0 0
    in
    isApplyIos is device.state.ios


isApplyIos : InjectorSentry -> List Sample -> InjectorSentry
isApplyIos is ios =
    case ios of
        x :: xs ->
            case x.sType of
                "armed" ->
                    isApplyIos { is | armed = floatToBool x.value } xs

                "inputIrrigator" ->
                    isApplyIos { is | inputIrrigator = floatToBool x.value } xs

                "inputWaterOn" ->
                    isApplyIos { is | inputWaterOn = floatToBool x.value } xs

                "inputInjector" ->
                    isApplyIos { is | inputInjector = floatToBool x.value } xs

                "gpioRelayInjectorEn" ->
                    isApplyIos { is | outputInjector = floatToBool x.value } xs

                "gpioShutdownEn" ->
                    isApplyIos { is | outputShutdown = floatToBool x.value } xs

                "flowRate" ->
                    isApplyIos { is | flowRate = x.value } xs

                "pressureMin" ->
                    isApplyIos { is | pressureMin = x.value } xs

                "pressureMax" ->
                    isApplyIos { is | pressureMax = x.value } xs

                "currentTankVolume" ->
                    isApplyIos { is | currentTankVolume = x.value } xs

                "tankCapacity" ->
                    isApplyIos { is | tankCapacity = x.value } xs

                "flowRateTarget" ->
                    isApplyIos { is | flowRateTarget = x.value } xs

                "flowWindowLow" ->
                    isApplyIos { is | flowWindowLow = x.value } xs

                "flowWindowHigh" ->
                    isApplyIos { is | flowWindowHigh = x.value } xs

                _ ->
                    isApplyIos is xs

        [] ->
            is


floatToBool : Float -> Bool
floatToBool input =
    input /= 0
