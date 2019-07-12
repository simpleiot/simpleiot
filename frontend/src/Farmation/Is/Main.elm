port module Main exposing (Msg(..), main, update, view)

import Array
import Bootstrap.Button as Button
import Bootstrap.ButtonGroup as ButtonGroup
import Bootstrap.Form as Form
import Bootstrap.Form.Input as Input
import Bootstrap.Grid as Grid
import Bootstrap.Grid.Col as Col
import Bootstrap.Utilities.Spacing as Spacing
import Browser
import Farmation.Is.Lcd as Lcd
import Farmation.Is.Outputs as Outputs
import Html exposing (Html, button, div, h2, h3, input, map, text)
import Html.Attributes exposing (class, placeholder, type_, value)
import Html.Events exposing (onClick, onInput)
import Json.Decode
import Json.Decode.Pipeline as Pipeline
import Json.Encode
import Sample exposing (..)
import Time


main =
    Browser.document
        { init = init
        , update = update
        , view = view
        , subscriptions = subscriptions
        }



-- ports


port portIn : (Json.Decode.Value -> msg) -> Sub msg


port portOut : Json.Encode.Value -> Cmd msg



-- Subscriptions


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ Time.every 1000 Tick
        , portIn (portValueDecoder >> ProcessPortValue)
        ]



-- MODEL


type alias SimInputs =
    { flowRate : Float
    , pressure : Float
    , gpioDigitalInjector : Bool
    , gpioDigitalIrrigator : Bool
    , gpioDigitalWaterOn : Bool
    , gpioDigitalIn : Bool
    , lindsayAcc1 : Bool
    , lindsayWaterOn : Bool
    , lindsayIrrigator : Bool
    }



-- system type defines must match Go state defines


systemTypeIS =
    0


systemTypeISSim =
    1


type alias State =
    { systemType : Int
    , gpioStatusLedRed : Bool
    , gpioStatusLedGreen : Bool
    , gpioRelayInjectorEn : Bool
    , gpioRelayShutdownEn : Bool
    , gpioRelayAuxEn : Bool
    , gpioDigitalInjector : Bool
    , gpioDigitalIrrigator : Bool
    , gpioDigitalWaterOn : Bool
    , gpioDigitalIn : Bool
    }


type alias Model =
    { lcdData : Lcd.Data
    , simInputs : SimInputs
    , state : State
    }


defaultState =
    State systemTypeIS False False False False False False False False False


defaultSimInputs =
    SimInputs 33 0 False False False False False False False


init : () -> ( Model, Cmd Msg )
init model =
    ( { lcdData = Lcd.defaultData
      , simInputs = defaultSimInputs
      , state = defaultState
      }
    , Cmd.none
    )



-- UPDATE


type Msg
    = ProcessPortValue (Result Json.Decode.Error PortValue)
    | Tick Time.Posix
    | GotLcdMsg Lcd.Msg
    | GotOutputsMsg Outputs.Msg
    | SimFlowRate String
    | SimPressure String
    | ButtonInj
    | ButtonIrg
    | ButtonWaterOn
    | ButtonDigIn
    | ButtonArm
    | ButtonLindsayWaterOn
    | ButtonLindsayAcc1
    | ButtonLindsayIrg


keyToSample : Lcd.Key -> Sample
keyToSample key =
    let
        keyString =
            case key of
                Lcd.KeyUp ->
                    "KeyUp"

                Lcd.KeyDown ->
                    "KeyDown"

                Lcd.KeyRight ->
                    "KeyRight"

                Lcd.KeyLeft ->
                    "KeyLeft"

                Lcd.KeyEnter ->
                    "KeyEnter"

                Lcd.KeySK1 ->
                    "KeySK1"

                Lcd.KeySK2 ->
                    "KeySK2"

                Lcd.KeySK3 ->
                    "KeySK3"

                Lcd.KeySK4 ->
                    "KeySK4"
    in
    Sample "key" keyString 0


processPortValue : PortValue -> Model -> ( Model, Cmd Msg )
processPortValue portValue model =
    case portValue of
        PixelValue pix ->
            ( { model
                | lcdData =
                    Lcd.setPixel pix.x pix.y pix.v model.lcdData
              }
            , Cmd.none
            )

        BltSolidValue blt ->
            ( { model
                | lcdData =
                    Lcd.setSolidBlock blt.x blt.y blt.w blt.h blt.v model.lcdData
              }
            , Cmd.none
            )

        BltValue blt ->
            ( { model
                | lcdData =
                    Lcd.setBlock blt.x blt.y blt.w blt.h blt.data model.lcdData
              }
            , Cmd.none
            )

        StateValue state ->
            ( { model | state = state }, Cmd.none )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        ProcessPortValue result ->
            case result of
                Ok portValue ->
                    processPortValue portValue model

                Err err ->
                    let
                        _ =
                            Debug.log "Port value decode error: " err
                    in
                    ( model, Cmd.none )

        GotLcdMsg lcdMsg ->
            case lcdMsg of
                Lcd.GotKey key ->
                    ( model
                    , keyToSample key
                        |> encodeSample
                        |> portOut
                    )

        GotOutputsMsg outputMsg ->
            case outputMsg of
                Outputs.NoOp ->
                    ( model, Cmd.none )

        SimFlowRate rate ->
            let
                simInputs =
                    model.simInputs

                rateF =
                    case String.toFloat rate of
                        Just val ->
                            val

                        Nothing ->
                            0

                newSimInputs =
                    { simInputs | flowRate = rateF }
            in
            ( { model | simInputs = newSimInputs }
            , Sample "simFlowRate" "" rateF
                |> encodeSample
                |> portOut
            )

        SimPressure rate ->
            let
                simInputs =
                    model.simInputs

                presF =
                    case String.toFloat rate of
                        Just val ->
                            val

                        Nothing ->
                            0

                newSimInputs =
                    { simInputs | pressure = presF }
            in
            ( { model | simInputs = newSimInputs }
            , Sample "simPressure" "" presF
                |> encodeSample
                |> portOut
            )

        ButtonInj ->
            let
                v =
                    not model.simInputs.gpioDigitalInjector

                vF =
                    if v then
                        1.0

                    else
                        0.0

                simInputs =
                    model.simInputs

                simInputsNew =
                    { simInputs | gpioDigitalInjector = v }
            in
            ( { model | simInputs = simInputsNew }
            , Sample "simGpioDigInj" "" vF
                |> encodeSample
                |> portOut
            )

        ButtonIrg ->
            let
                v =
                    not model.simInputs.gpioDigitalIrrigator

                vF =
                    if v then
                        1.0

                    else
                        0.0

                simInputs =
                    model.simInputs

                simInputsNew =
                    { simInputs | gpioDigitalIrrigator = v }
            in
            ( { model | simInputs = simInputsNew }
            , Sample "simGpioDigIrg" "" vF
                |> encodeSample
                |> portOut
            )

        ButtonWaterOn ->
            let
                v =
                    not model.simInputs.gpioDigitalWaterOn

                vF =
                    if v then
                        1.0

                    else
                        0.0

                simInputs =
                    model.simInputs

                simInputsNew =
                    { simInputs | gpioDigitalWaterOn = v }
            in
            ( { model | simInputs = simInputsNew }
            , Sample "simGpioDigWaterOn" "" vF
                |> encodeSample
                |> portOut
            )

        ButtonDigIn ->
            let
                v =
                    not model.simInputs.gpioDigitalIn

                vF =
                    if v then
                        1.0

                    else
                        0.0

                simInputs =
                    model.simInputs

                simInputsNew =
                    { simInputs | gpioDigitalIn = v }
            in
            ( { model | simInputs = simInputsNew }
            , Sample "simGpioDigIn" "" vF
                |> encodeSample
                |> portOut
            )

        ButtonLindsayWaterOn ->
            let
                v =
                    not model.simInputs.lindsayWaterOn

                vF =
                    if v then
                        1.0

                    else
                        0.0

                simInputs =
                    model.simInputs

                simInputsNew =
                    { simInputs | lindsayWaterOn = v }
            in
            ( { model | simInputs = simInputsNew }
            , Sample "simLindsayWaterOn" "" vF
                |> encodeSample
                |> portOut
            )

        ButtonLindsayAcc1 ->
            let
                v =
                    not model.simInputs.lindsayAcc1

                vF =
                    if v then
                        1.0

                    else
                        0.0

                simInputs =
                    model.simInputs

                simInputsNew =
                    { simInputs | lindsayAcc1 = v }
            in
            ( { model | simInputs = simInputsNew }
            , Sample "simLindsayAcc1" "" vF
                |> encodeSample
                |> portOut
            )

        ButtonLindsayIrg ->
            let
                v =
                    not model.simInputs.lindsayIrrigator

                vF =
                    if v then
                        1.0

                    else
                        0.0

                simInputs =
                    model.simInputs

                simInputsNew =
                    { simInputs | lindsayIrrigator = v }
            in
            ( { model | simInputs = simInputsNew }
            , Sample "simLindsayIrrigator" "" vF
                |> encodeSample
                |> portOut
            )

        ButtonArm ->
            ( model
            , Sample "simArm" "" 0
                |> encodeSample
                |> portOut
            )

        Tick _ ->
            ( model, Cmd.none )



-- VIEW


buttonType : Bool -> Button.Option msg
buttonType on =
    if on then
        Button.primary

    else
        Button.secondary


renderSimOutputs : State -> Html Msg
renderSimOutputs state =
    div []
        [ Grid.row []
            [ Grid.col [ Col.xs12, Col.sm6, Col.md5 ]
                [ map GotOutputsMsg
                    (Outputs.statusLed state.gpioStatusLedRed
                        state.gpioStatusLedGreen
                    )
                ]
            ]
        , Grid.row
            []
            [ Grid.col [ Col.xs12, Col.sm6, Col.md5 ]
                [ map GotOutputsMsg
                    (Outputs.relay "Inj" state.gpioRelayInjectorEn)
                , map GotOutputsMsg
                    (Outputs.relay "Shutdn" state.gpioRelayShutdownEn)
                , map GotOutputsMsg
                    (Outputs.relay "Aux" state.gpioRelayAuxEn)
                ]
            ]
        ]


renderDigitalInputs : State -> Html Msg
renderDigitalInputs state =
    div []
        [ Grid.row
            []
            [ Grid.col [ Col.xs12, Col.sm6, Col.md5 ]
                [ map GotOutputsMsg
                    (Outputs.relay "Inj" state.gpioDigitalInjector)
                , map GotOutputsMsg
                    (Outputs.relay "Irr" state.gpioDigitalIrrigator)
                , map GotOutputsMsg
                    (Outputs.relay "Water" state.gpioDigitalWaterOn)
                , map GotOutputsMsg
                    (Outputs.relay "In" state.gpioDigitalIn)
                ]
            ]
        ]


renderLindsaySimInputs : SimInputs -> Html Msg
renderLindsaySimInputs inputs =
    Grid.row []
        [ Grid.col [ Col.xs12, Col.sm6, Col.md5 ]
            [ div []
                [ Button.button
                    [ buttonType inputs.lindsayAcc1
                    , Button.attrs
                        [ Spacing.m1
                        , onClick ButtonLindsayAcc1
                        ]
                    ]
                    [ text "Lindsay Acc1" ]
                , Button.button
                    [ buttonType inputs.lindsayWaterOn
                    , Button.attrs
                        [ Spacing.m1
                        , onClick ButtonLindsayWaterOn
                        ]
                    ]
                    [ text "Lindsay Water On" ]
                , Button.button
                    [ buttonType inputs.lindsayIrrigator
                    , Button.attrs
                        [ Spacing.m1
                        , onClick ButtonLindsayIrg
                        ]
                    ]
                    [ text "Lindsay Irrigator" ]
                ]
            ]
        ]


renderSimInputs : SimInputs -> Html Msg
renderSimInputs inputs =
    Grid.row []
        [ Grid.col [ Col.xs12, Col.sm6, Col.md5 ]
            [ Form.group []
                [ Form.label [] [ text "Sim flow rate" ]
                , Input.text
                    [ Input.attrs
                        [ placeholder "enter flow rate"
                        , onInput SimFlowRate
                        , value (String.fromFloat inputs.flowRate)
                        , type_ "number"
                        ]
                    ]
                ]
            , Form.group []
                [ Form.label [] [ text "Sim pressure" ]
                , Input.text
                    [ Input.attrs
                        [ placeholder "enter pressure"
                        , onInput SimPressure
                        , value (String.fromFloat inputs.pressure)
                        , type_ "number"
                        ]
                    ]
                ]
            , div []
                [ Button.button
                    [ buttonType inputs.gpioDigitalInjector
                    , Button.attrs
                        [ Spacing.m1
                        , onClick ButtonInj
                        ]
                    ]
                    [ text "injector" ]
                , Button.button
                    [ buttonType inputs.gpioDigitalIrrigator
                    , Button.attrs
                        [ Spacing.m1
                        , onClick ButtonIrg
                        ]
                    ]
                    [ text "irrigator" ]
                , Button.button
                    [ buttonType inputs.gpioDigitalWaterOn
                    , Button.attrs
                        [ Spacing.m1
                        , onClick ButtonWaterOn
                        ]
                    ]
                    [ text "water on" ]
                , Button.button
                    [ buttonType inputs.gpioDigitalIn
                    , Button.attrs
                        [ Spacing.m1
                        , onClick ButtonDigIn
                        ]
                    ]
                    [ text "digital in" ]
                ]
            , Button.button
                [ Button.secondary
                , Button.attrs
                    [ Spacing.m1
                    , onClick ButtonArm
                    ]
                ]
                [ text "Arm" ]
            ]
        ]


view : Model -> Browser.Document Msg
view model =
    let
        simInputs =
            if model.state.systemType == systemTypeISSim then
                div []
                    [ h3 [] [ text "Sim Inputs" ]
                    , renderSimInputs model.simInputs
                    , renderLindsaySimInputs model.simInputs
                    ]

            else
                div []
                    [ h3 [] [ text "Digital Inputs" ]
                    , renderDigitalInputs model.state
                    ]
    in
    { title = "Injector • Sentry"
    , body =
        [ div []
            [ h2 [] [ text "Injector Sentry" ]
            , map GotLcdMsg (Lcd.lcd model.lcdData)
            ]
        , Grid.container []
            [ simInputs
            , h3 [] [ text "Sim Outputs" ]
            , renderSimOutputs model.state
            ]
        ]
    }



-- Misc functions/data structures


type alias Pixel =
    { x : Int
    , y : Int
    , v : Bool
    }


stateDecoder : Json.Decode.Decoder State
stateDecoder =
    Json.Decode.succeed State
        |> Pipeline.required "systemType" Json.Decode.int
        |> Pipeline.required "gpioStatusLedRed" Json.Decode.bool
        |> Pipeline.required "gpioStatusLedGreen" Json.Decode.bool
        |> Pipeline.required "gpioRelayInjectorEn" Json.Decode.bool
        |> Pipeline.required "gpioRelayShutdownEn" Json.Decode.bool
        |> Pipeline.required "gpioRelayAuxEn" Json.Decode.bool
        |> Pipeline.required "gpioDigitalInjector" Json.Decode.bool
        |> Pipeline.required "gpioDigitalIrrigator" Json.Decode.bool
        |> Pipeline.required "gpioDigitalWaterOn" Json.Decode.bool
        |> Pipeline.required "gpioDigitalIn" Json.Decode.bool


pixelDecoder : Json.Decode.Decoder Pixel
pixelDecoder =
    Json.Decode.map3 Pixel
        (Json.Decode.field "x" Json.Decode.int)
        (Json.Decode.field "y" Json.Decode.int)
        (Json.Decode.field "v" Json.Decode.bool)


type alias BltSolid =
    { x : Int
    , y : Int
    , w : Int
    , h : Int
    , v : Bool
    }


bltSolidDecoder : Json.Decode.Decoder BltSolid
bltSolidDecoder =
    Json.Decode.map5 BltSolid
        (Json.Decode.field "x" Json.Decode.int)
        (Json.Decode.field "y" Json.Decode.int)
        (Json.Decode.field "w" Json.Decode.int)
        (Json.Decode.field "h" Json.Decode.int)
        (Json.Decode.field "v" Json.Decode.bool)


type alias Blt =
    { x : Int
    , y : Int
    , w : Int
    , h : Int
    , data : Array.Array Bool
    }


bltDecoder : Json.Decode.Decoder Blt
bltDecoder =
    Json.Decode.map5 Blt
        (Json.Decode.field "x" Json.Decode.int)
        (Json.Decode.field "y" Json.Decode.int)
        (Json.Decode.field "w" Json.Decode.int)
        (Json.Decode.field "h" Json.Decode.int)
        (Json.Decode.field "data" (Json.Decode.array Json.Decode.bool))


type PortValue
    = PixelValue Pixel
    | BltSolidValue BltSolid
    | BltValue Blt
    | StateValue State


portDecoder : Json.Decode.Decoder PortValue
portDecoder =
    Json.Decode.oneOf
        [ Json.Decode.map BltSolidValue bltSolidDecoder
        , Json.Decode.map PixelValue pixelDecoder
        , Json.Decode.map BltValue bltDecoder
        , Json.Decode.map StateValue stateDecoder
        ]


portValueDecoder : Json.Decode.Value -> Result Json.Decode.Error PortValue
portValueDecoder v =
    Json.Decode.decodeValue portDecoder v
