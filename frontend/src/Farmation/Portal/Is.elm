module Farmation.Portal.Is exposing (view)

import Data.Device
import Data.Sample exposing (Sample)
import Element exposing (..)
import Element.Border as Border
import Element.Input as Input
import Round
import UI.Icon as Icon
import UI.Style as Style


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


view :
    { msgUpdateDesc : String -> String -> msg
    , msgDiscardUpdate : msg
    , msgSave : String -> Data.Device.Config -> msg
    , msgDeleteDevice : String -> msg
    , device : Data.Device.Device
    , mod : Bool
    , isRoot : Bool
    }
    -> Element msg
view { msgUpdateDesc, msgDiscardUpdate, msgSave, msgDeleteDevice, device, mod, isRoot } =
    let
        is =
            deviceToInjectorSentry device

        inputInj l =
            if is.inputInjector then
                l ++ [ image [] { src = "public/Injector.png", description = "injector" } ]

            else
                l

        inputWater l =
            if is.inputWaterOn then
                l ++ [ image [] { src = "public/WaterOn.png", description = "water on" } ]

            else
                l

        inputIrr l =
            if is.inputIrrigator then
                l ++ [ image [] { src = "public/Irrigator.png", description = "irrigator" } ]

            else
                l

        flow l =
            l ++ [ el Style.h3 <| text (Round.round 1 is.flowRate ++ " GPH") ]

        armed l =
            if is.armed then
                l ++ [ image [] { src = "public/Armed.png", description = "armed" } ]

            else
                l

        outputInj l =
            if is.outputInjector then
                l ++ [ image [] { src = "public/Injector.png", description = "injector" } ]

            else
                l

        outputShutdown l =
            if is.outputShutdown then
                l ++ [ image [] { src = "public/Shutdown.png", description = "shutdown" } ]

            else
                l

        statusElements =
            inputInj []
                |> inputWater
                |> inputIrr
                |> flow
                |> armed
                |> outputInj
                |> outputShutdown
    in
    column
        [ padding 20
        , spacing 20
        , width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        ]
        [ wrappedRow [ spacing 10 ]
            [ el Style.h2 <| text device.id
            , if isRoot then
                Icon.x (msgDeleteDevice device.id)

              else
                Element.none
            , Input.text
                Style.h3
                { label = Input.labelHidden "device description"
                , text = device.config.description
                , placeholder = Just <| Input.placeholder [] <| text "device description"
                , onChange = \d -> msgUpdateDesc device.id d
                }
            , if mod then
                Icon.check (msgSave device.id device.config)

              else
                Element.none
            , if mod then
                Icon.x msgDiscardUpdate

              else
                Element.none
            ]
        , wrappedRow [ spacing 20 ] statusElements
        , text ("Min Pressure: " ++ Round.round 0 is.pressureMin)
        , text ("Max Pressure: " ++ Round.round 0 is.pressureMax)
        , text
            ("Tank Level: "
                ++ Round.round 0
                    is.currentTankVolume
                ++ " of "
                ++ Round.round 0 is.tankCapacity
                ++ " gal"
            )
        , text ("Target Flow: " ++ Round.round 0 is.flowRateTarget)
        , text
            ("Target Flow Window: "
                ++ Round.round 0 is.flowWindowLow
                ++ " to "
                ++ Round.round 0 is.flowWindowHigh
            )
        ]


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

                "gpioRelayShutdownEn" ->
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
